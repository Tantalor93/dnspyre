package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync/atomic"

	"github.com/miekg/dns"
	"github.com/quic-go/quic-go"
)

type doqHandler func(req *dns.Msg) *dns.Msg

// doqServer is a DoQ test DNS server.
type doqServer struct {
	addr     string
	listener *quic.Listener
	closed   atomic.Bool
	handler  doqHandler
}

func newDoQServer(f doqHandler) *doqServer {
	server := doqServer{handler: f}
	return &server
}

func (d *doqServer) start() {
	listener, err := quic.ListenAddr("localhost:0", generateTLSConfig(), nil)
	if err != nil {
		panic(err)
	}
	d.listener = listener
	d.addr = listener.Addr().String()
	go func() {
		for {
			conn, err := listener.Accept(context.Background())
			if err != nil {
				if !d.closed.Load() {
					panic(err)
				}
				return
			}

			go func() {
				for {
					stream, err := conn.AcceptStream(context.Background())
					if err != nil {
						return
					}

					req, err := readDOQMessage(stream)
					if err != nil {
						return
					}

					resp := d.handler(req)
					if resp == nil {
						// this should cause timeout
						return
					}
					pack, err := resp.Pack()
					if err != nil {
						return
					}
					packWithPrefix := make([]byte, 2+len(pack))
					binary.BigEndian.PutUint16(packWithPrefix, uint16(len(pack)))
					copy(packWithPrefix[2:], pack)
					_, _ = stream.Write(packWithPrefix)
					_ = stream.Close()
				}
			}()
		}
	}()
}

func (d *doqServer) stop() {
	if !d.closed.Swap(true) {
		_ = d.listener.Close()
	}
}

func generateTLSConfig() *tls.Config {
	cert, err := tls.LoadX509KeyPair("testdata/test.crt", "testdata/test.key")
	if err != nil {
		panic(err)
	}

	certs, err := os.ReadFile("testdata/test.crt")
	if err != nil {
		panic(err)
	}

	pool, err := x509.SystemCertPool()
	if err != nil {
		panic(err)
	}
	pool.AppendCertsFromPEM(certs)

	return &tls.Config{
		ServerName:   "localhost",
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"doq"},
		RootCAs:      pool,
		MinVersion:   tls.VersionTLS12,
	}
}

func readDOQMessage(r io.Reader) (*dns.Msg, error) {
	// All DNS messages (queries and responses) sent over DoQ connections MUST
	// be encoded as a 2-octet length field followed by the message content as
	// specified in [RFC1035].
	// See https://www.rfc-editor.org/rfc/rfc9250.html#section-4.2-4
	sizeBuf := make([]byte, 2)
	_, err := io.ReadFull(r, sizeBuf)
	if err != nil {
		return nil, err
	}

	size := binary.BigEndian.Uint16(sizeBuf)

	if size == 0 {
		return nil, fmt.Errorf("message size is 0: probably unsupported DoQ version")
	}

	buf := make([]byte, size)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}

	// A client or server receives a STREAM FIN before receiving all the bytes
	// for a message indicated in the 2-octet length field.
	// See https://www.rfc-editor.org/rfc/rfc9250#section-4.3.3-2.2
	if size != uint16(len(buf)) {
		return nil, fmt.Errorf("message size does not match 2-byte prefix")
	}

	msg := &dns.Msg{}
	err = msg.Unpack(buf)

	return msg, err
}
