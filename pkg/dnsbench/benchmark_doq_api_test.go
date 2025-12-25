package dnsbench_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/quic-go/quic-go"
	"github.com/stretchr/testify/suite"
	"github.com/tantalor93/dnspyre/v3/pkg/dnsbench"
)

type DoQTestSuite struct {
	suite.Suite
}

func TestDoQTestSuite(t *testing.T) {
	suite.Run(t, new(DoQTestSuite))
}

func (suite *DoQTestSuite) TestBenchmark_Run() {
	server := newDoQServer(func(_ *quic.Conn, r *dns.Msg) *dns.Msg {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)
		return ret
	})
	server.start()
	defer server.stop()

	buf := bytes.Buffer{}
	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A", "AAAA"},
		Server:         "quic://" + server.addr,
		TCP:            true,
		Concurrency:    2,
		Count:          1,
		Probability:    1,
		WriteTimeout:   1 * time.Second,
		ReadTimeout:    3 * time.Second,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
		Rcodes:         true,
		Recurse:        true,
		Insecure:       true,
		Writer:         &buf,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	suite.Require().NoError(err, "expected no error from benchmark run")
	assertResult(suite.T(), rs)
	suite.Equal(fmt.Sprintf("Using 1 hostnames\nBenchmarking %s via quic with 2 concurrent requests \n", server.addr), buf.String())
}

func (suite *DoQTestSuite) TestBenchmark_Run_separate_connections() {
	tests := []struct {
		name                    string
		separateConnections     bool
		wantNumberOfConnections int
	}{
		{
			name:                    "separate connections",
			separateConnections:     true,
			wantNumberOfConnections: 5,
		},
		{
			name:                    "shared connections",
			separateConnections:     false,
			wantNumberOfConnections: 1,
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			mutex := sync.Mutex{}
			remoteAddrs := make(map[string]int)

			server := newDoQServer(func(c *quic.Conn, r *dns.Msg) *dns.Msg {
				mutex.Lock()
				remoteAddrs[c.RemoteAddr().String()]++
				mutex.Unlock()

				ret := new(dns.Msg)
				ret.SetReply(r)
				ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))
				return ret
			})
			server.start()
			defer server.stop()

			buf := bytes.Buffer{}
			bench := dnsbench.Benchmark{
				Queries:                   []string{"example.org"},
				Types:                     []string{"A"},
				Server:                    "quic://" + server.addr,
				TCP:                       true,
				Concurrency:               5,
				Count:                     2,
				Probability:               1,
				WriteTimeout:              1 * time.Second,
				ReadTimeout:               3 * time.Second,
				ConnectTimeout:            1 * time.Second,
				RequestTimeout:            5 * time.Second,
				Rcodes:                    true,
				Recurse:                   true,
				DohMethod:                 dnsbench.PostHTTPMethod,
				Writer:                    &buf,
				SeparateWorkerConnections: tt.separateConnections,
				Insecure:                  true,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			rs, err := bench.Run(ctx)

			// stop right away to mitigate race detector failures
			server.stop()

			suite.Require().NoError(err, "expected no error from benchmark run")
			suite.Require().Len(rs, 5)
			for _, v := range rs {
				suite.Empty(v.Errors)
			}
			suite.Len(remoteAddrs, tt.wantNumberOfConnections)
			suite.Equal(fmt.Sprintf("Using 1 hostnames\nBenchmarking %s via quic with 5 concurrent requests \n", server.addr), buf.String())
		})
	}
}

func (suite *DoQTestSuite) TestBenchmark_Run_truncated() {
	server := newDoQServer(func(_ *quic.Conn, r *dns.Msg) *dns.Msg {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))
		ret.Truncated = true

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)
		return ret
	})
	server.start()
	defer server.stop()

	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A", "AAAA"},
		Server:         "quic://" + server.addr,
		TCP:            true,
		Concurrency:    2,
		Count:          1,
		Probability:    1,
		WriteTimeout:   1 * time.Second,
		ReadTimeout:    3 * time.Second,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
		Rcodes:         true,
		Recurse:        true,
		Insecure:       true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	suite.Require().NoError(err, "expected no error from benchmark run")
	suite.Require().Len(rs, 2, "expected results from two workers")

	suite.EqualValues(2, rs[0].Counters.Total, "there should be executions")
	suite.EqualValues(2, rs[0].Counters.Truncated, "there should be truncated messages")
	suite.EqualValues(2, rs[1].Counters.Total, "there should be executions")
	suite.EqualValues(2, rs[1].Counters.Truncated, "there should be truncated messages")
}

func (suite *DoQTestSuite) TestBenchmark_Run_error() {
	server := newDoQServer(func(_ *quic.Conn, _ *dns.Msg) *dns.Msg {
		return nil
	})
	server.start()
	defer server.stop()

	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A", "AAAA"},
		Server:         "quic://" + server.addr,
		TCP:            true,
		Concurrency:    2,
		Count:          1,
		Probability:    1,
		WriteTimeout:   100 * time.Millisecond,
		ReadTimeout:    300 * time.Millisecond,
		ConnectTimeout: 100 * time.Millisecond,
		RequestTimeout: 500 * time.Millisecond,
		Rcodes:         true,
		Recurse:        true,
		Insecure:       true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	suite.Require().NoError(err, "expected no error from benchmark run")
	suite.Require().Len(rs, 2, "expected results from two workers")

	suite.EqualValues(2, rs[0].Counters.Total, "there should be executions")
	suite.EqualValues(2, rs[0].Counters.IOError, "there should be errors")
	suite.EqualValues(2, rs[1].Counters.Total, "there should be executions")
	suite.EqualValues(2, rs[1].Counters.IOError, "there should be errors")
}

type doqHandler func(conn *quic.Conn, req *dns.Msg) *dns.Msg

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

					resp := d.handler(conn, req)
					if resp == nil {
						// this should cause timeout
						return
					}
					pack, err := resp.Pack()
					if err != nil {
						return
					}
					packWithPrefix := make([]byte, 2+len(pack))
					// nolint:gosec
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
	// nolint:gosec
	if size != uint16(len(buf)) {
		return nil, fmt.Errorf("message size does not match 2-byte prefix")
	}

	msg := &dns.Msg{}
	err = msg.Unpack(buf)

	return msg, err
}
