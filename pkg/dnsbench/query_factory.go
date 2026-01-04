package dnsbench

import (
	"context"
	"crypto/tls"
	"log"
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

	// Set up dialer with source IP if specified
	var dialer *net.Dialer
	if b.SourceIP != "" {
		localAddr, err := net.ResolveTCPAddr("tcp", b.SourceIP+":0")
		if err != nil {
			// This should not happen as source IP is validated in init()
			log.Printf("Warning: failed to resolve source IP %s for DoH: %v", b.SourceIP, err)
		} else {
			dialer = &net.Dialer{
				LocalAddr: localAddr,
				Timeout:   b.ConnectTimeout,
			}
		}
	}

	switch b.DohProtocol {
	case HTTP3Proto:
		// nolint:gosec
		// HTTP3 doesn't support custom dialer with source IP in a straightforward way
		tr = &http3.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: b.Insecure}}
	case HTTP2Proto:
		// nolint:gosec
		h2Transport := &http2.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: b.Insecure}}
		if dialer != nil {
			h2Transport.DialTLS = func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				// Use "tcp" network for the underlying connection
				conn, err := dialer.Dial("tcp", addr)
				if err != nil {
					return nil, err
				}
				tlsConn := tls.Client(conn, cfg)
				if err := tlsConn.Handshake(); err != nil {
					conn.Close()
					return nil, err
				}
				return tlsConn, nil
			}
		}
		tr = h2Transport
	case HTTP1Proto:
		fallthrough
	default:
		// nolint:gosec
		h1Transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: b.Insecure}}
		if dialer != nil {
			h1Transport.DialContext = dialer.DialContext
		}
		tr = h1Transport
	}
	c := http.Client{Transport: tr, Timeout: b.ReadTimeout}
	dohClient := doh.NewClient(b.Server, doh.WithHTTPClient(&c))

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

	// Set up custom dialer if source IP is specified
	if b.SourceIP != "" {
		// Determine which address type to use based on network
		var localAddr net.Addr
		var err error
		
		if network == UDPTransport {
			localAddr, err = net.ResolveUDPAddr("udp", b.SourceIP+":0")
		} else {
			// For TCP and TLS (tcp-tls), use TCP address
			localAddr, err = net.ResolveTCPAddr("tcp", b.SourceIP+":0")
		}
		
		if err != nil {
			// This should not happen as source IP is validated in init()
			log.Printf("Warning: failed to resolve source IP %s for DNS: %v", b.SourceIP, err)
		} else {
			client.Dialer = &net.Dialer{
				LocalAddr: localAddr,
				Timeout:   b.ConnectTimeout,
			}
		}
	}

	return client
}
