package dnsbench

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"github.com/miekg/dns"
	"github.com/quic-go/quic-go/http3"
	"github.com/tantalor93/doh-go/doh"
	"github.com/tantalor93/doq-go/doq"
	"golang.org/x/net/http2"
)

func workerQueryFactory(b *Benchmark) func() queryFunc {
	switch {
	case b.useDoH:
		return dohQueryFactory(b)
	case b.useQuic:
		return doqQueryFactory(b)
	default:
		return dnsQueryFactory(b)
	}
}

func dnsQueryFactory(b *Benchmark) func() queryFunc {
	return func() queryFunc {
		dnsClient := getDNSClient(b)
		var co *dns.Conn
		var i int64
		// this allows DoT and plain DNS protocols to support counting queries per connection
		// and granular control of the connection
		return func(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
			if co != nil && b.QperConn > 0 && i%b.QperConn == 0 {
				co.Close()
				co = nil
			}
			i++
			if co == nil {
				var err error
				co, err = dnsClient.DialContext(ctx, b.Server)
				if err != nil {
					return nil, err
				}
			}
			r, _, err := dnsClient.ExchangeWithConnContext(ctx, msg, co)
			if err != nil {
				co.Close()
				co = nil
				return nil, err
			}
			return r, nil
		}
	}
}

func doqQueryFactory(b *Benchmark) func() queryFunc {
	if b.SeparateWorkerConnections {
		return func() queryFunc {
			quicClient := getDoQClient(b)
			return quicClient.Send
		}
	}
	quicClient := getDoQClient(b)
	return func() queryFunc {
		return quicClient.Send
	}
}

func dohQueryFactory(b *Benchmark) func() queryFunc {
	if b.SeparateWorkerConnections {
		return func() queryFunc {
			return dohQuery(b)
		}
	}
	dohQuery := dohQuery(b)
	return func() queryFunc {
		return dohQuery
	}
}

func dohQuery(b *Benchmark) queryFunc {
	var tr http.RoundTripper

	// Create custom dialer if local address is specified
	var dialer *net.Dialer
	if len(b.LocalAddr) > 0 {
		localAddr, err := net.ResolveTCPAddr("tcp", b.LocalAddr)
		if err == nil {
			dialer = &net.Dialer{
				LocalAddr: localAddr,
			}
		}
	}
	if dialer == nil {
		dialer = &net.Dialer{}
	}

	switch b.DohProtocol {
	case HTTP3Proto:
		// nolint:gosec
		tr = &http3.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: b.Insecure}}
	case HTTP2Proto:
		// nolint:gosec
		tr = &http2.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: b.Insecure},
			DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
				conn, err := dialer.DialContext(ctx, network, addr)
				if err != nil {
					return nil, err
				}
				tlsConn := tls.Client(conn, cfg)
				if err := tlsConn.HandshakeContext(ctx); err != nil {
					conn.Close()
					return nil, err
				}
				return tlsConn, nil
			},
		}
	case HTTP1Proto:
		fallthrough
	default:
		// nolint:gosec
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: b.Insecure},
			DialContext:     dialer.DialContext,
		}
	}
	c := http.Client{Transport: tr, Timeout: b.ReadTimeout}
	dohClient := doh.NewClient(b.Server, doh.WithHTTPClient(&c), doh.WithUserAgent(b.DohUserAgent))

	switch b.DohMethod {
	case PostHTTPMethod:
		return dohClient.SendViaPost
	case GetHTTPMethod:
		return dohClient.SendViaGet
	default:
		return dohClient.SendViaPost
	}
}

func getDoQClient(b *Benchmark) *doq.Client {
	h, _, _ := net.SplitHostPort(b.Server)

	// Note: DoQ client doesn't currently support custom local address binding
	// This would require upstream library support
	return doq.NewClient(b.Server,
		// nolint:gosec
		doq.WithTLSConfig(&tls.Config{ServerName: h, InsecureSkipVerify: b.Insecure}),
		doq.WithReadTimeout(b.ReadTimeout),
		doq.WithWriteTimeout(b.WriteTimeout),
		doq.WithConnectTimeout(b.ConnectTimeout),
	)
}

func getDNSClient(b *Benchmark) *dns.Client {
	network := UDPTransport
	if b.TCP {
		network = TCPTransport
	}
	if b.DOT {
		network = TLSTransport
	}

	client := &dns.Client{
		Net:          network,
		DialTimeout:  b.ConnectTimeout,
		WriteTimeout: b.WriteTimeout,
		ReadTimeout:  b.ReadTimeout,
		Timeout:      b.RequestTimeout,
		// nolint:gosec
		TLSConfig: &tls.Config{InsecureSkipVerify: b.Insecure},
	}

	// Set up custom dialer if local address is specified
	if len(b.LocalAddr) > 0 {
		var localAddr net.Addr
		var err error

		// Determine the appropriate address type based on the network
		if network == UDPTransport {
			localAddr, err = net.ResolveUDPAddr("udp", b.LocalAddr)
		} else {
			localAddr, err = net.ResolveTCPAddr("tcp", b.LocalAddr)
		}

		if err == nil {
			client.Dialer = &net.Dialer{
				LocalAddr: localAddr,
				Timeout:   b.ConnectTimeout,
			}
		}
	}

	return client
}
