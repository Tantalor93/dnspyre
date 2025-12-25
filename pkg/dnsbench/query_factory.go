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
	switch b.DohProtocol {
	case HTTP3Proto:
		// nolint:gosec
		tr = &http3.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: b.Insecure}}
	case HTTP2Proto:
		// nolint:gosec
		tr = &http2.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: b.Insecure}}
	case HTTP1Proto:
		fallthrough
	default:
		// nolint:gosec
		tr = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: b.Insecure}}
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

	return &dns.Client{
		Net:          network,
		DialTimeout:  b.ConnectTimeout,
		WriteTimeout: b.WriteTimeout,
		ReadTimeout:  b.ReadTimeout,
		Timeout:      b.RequestTimeout,
		// nolint:gosec
		TLSConfig: &tls.Config{InsecureSkipVerify: b.Insecure},
	}
}
