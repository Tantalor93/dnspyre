package dnsbench

import (
	"context"
	"crypto/tls"
	"math/rand"
	"net"
	"net/http"
	"time"

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
		// create a new lock free rand source for this goroutine (for CIDR random IP generation)
		// nolint:gosec
		rando := rand.New(rand.NewSource(time.Now().UnixNano()))

		// this allows DoT and plain DNS protocols to support counting queries per connection
		// and granular control of the connection
		return func(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
			if co != nil && b.QperConn > 0 && i%b.QperConn == 0 {
				co.Close()
				co = nil
			}
			i++
			if co == nil {
				// Update local address if using CIDR
				if b.localAddrNet != nil && dnsClient.Dialer != nil {
					randomIP := randomIPFromCIDR(b.localAddrNet, rando)
					network := dnsClient.Net
					if network == UDPTransport {
						dnsClient.Dialer.LocalAddr = &net.UDPAddr{IP: randomIP}
					} else {
						dnsClient.Dialer.LocalAddr = &net.TCPAddr{IP: randomIP}
					}
				}

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
	// nolint:gosec
	rando := rand.New(rand.NewSource(time.Now().UnixNano()))

	var dialContext func(ctx context.Context, network, addr string) (net.Conn, error)

	if b.localAddrIP != nil {
		if b.localAddrNet != nil {
			// CIDR mode: create a custom dial function that generates random IPs for each connection
			dialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				randomIP := randomIPFromCIDR(b.localAddrNet, rando)
				dialer := &net.Dialer{
					LocalAddr: &net.TCPAddr{IP: randomIP},
				}
				return dialer.DialContext(ctx, network, addr)
			}
		} else {
			// Single IP mode
			dialer := &net.Dialer{
				LocalAddr: &net.TCPAddr{IP: b.localAddrIP},
			}
			dialContext = dialer.DialContext
		}
	} else {
		// No local address specified
		dialer := &net.Dialer{}
		dialContext = dialer.DialContext
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
				conn, err := dialContext(ctx, network, addr)
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
			DialContext:     dialContext,
		}
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
	if b.localAddrIP != nil {
		var localAddr net.Addr

		// Determine the appropriate address type based on the network
		if network == UDPTransport {
			localAddr = &net.UDPAddr{IP: b.localAddrIP}
		} else {
			localAddr = &net.TCPAddr{IP: b.localAddrIP}
		}

		client.Dialer = &net.Dialer{
			LocalAddr: localAddr,
			Timeout:   b.ConnectTimeout,
		}
	}

	return client
}
