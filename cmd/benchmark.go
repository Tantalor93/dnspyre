package cmd

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/fatih/color"
	"github.com/miekg/dns"
	"github.com/quic-go/quic-go/http3"
	"github.com/tantalor93/doh-go/doh"
	"github.com/tantalor93/doq-go/doq"
	"go.uber.org/ratelimit"
	"golang.org/x/net/http2"
)

const dnsTimeout = time.Second * 4

var client = http.Client{
	Timeout: 120 * time.Second,
}

// Benchmark is representation of benchmark scenario.
type Benchmark struct {
	Server      string
	Types       []string
	Count       int64
	Concurrency uint32

	Rate     int
	QperConn int64

	Recurse bool

	Probability float64

	UDPSize uint16
	EdnsOpt string

	TCP bool
	DOT bool

	WriteTimeout time.Duration
	ReadTimeout  time.Duration

	Rcodes bool

	HistDisplay bool
	HistMin     time.Duration
	HistMax     time.Duration
	HistPre     int

	Csv  string
	JSON bool

	Silent bool
	Color  bool

	PlotDir    string
	PlotFormat string

	DohMethod   string
	DohProtocol string

	Insecure bool

	Queries []string

	Duration time.Duration

	// internal variable so we do not have to parse the address with each request.
	useDoH  bool
	useQuic bool
}

type queryFunc func(context.Context, string, *dns.Msg) (*dns.Msg, error)

func (b *Benchmark) normalize() error {
	b.useDoH, _ = isHTTPUrl(b.Server)
	b.useQuic = strings.HasPrefix(b.Server, "quic://")
	if b.useQuic {
		b.Server = strings.TrimPrefix(b.Server, "quic://")
	}

	b.addPortIfMissing()

	if b.Count == 0 && b.Duration == 0 {
		b.Count = 1
	}

	if b.Duration > 0 && b.Count > 0 {
		return errors.New("--number and --duration is specified at once, only one can be used")
	}
	return nil
}

// Run executes benchmark, if benchmark is unable to start the error is returned, otherwise array of results from parallel benchmark goroutines is returned.
func (b *Benchmark) Run(ctx context.Context) ([]*ResultStats, error) {
	if err := b.normalize(); err != nil {
		return nil, err
	}

	color.NoColor = !b.Color

	var questions []string
	for _, q := range b.Queries {
		if ok, _ := isHTTPUrl(q); ok {
			resp, err := client.Get(q)
			if err != nil {
				return nil, fmt.Errorf("failed to download file '%s' with error '%v'", q, err)
			}
			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				return nil, fmt.Errorf("failed to download file '%s' with status '%s'", q, resp.Status)
			}
			scanner := bufio.NewScanner(resp.Body)
			for scanner.Scan() {
				questions = append(questions, dns.Fqdn(scanner.Text()))
			}
		} else {
			questions = append(questions, dns.Fqdn(q))
		}
	}

	if b.Duration != 0 {
		timeoutCtx, cancel := context.WithTimeout(ctx, b.Duration)
		ctx = timeoutCtx
		defer cancel()
	}

	if !b.Silent && !b.JSON {
		fmt.Printf("Using %s hostnames\n", highlightStr(len(questions)))
	}

	var qTypes []uint16
	for _, v := range b.Types {
		qTypes = append(qTypes, dns.StringToType[v])
	}

	network := "udp"
	if b.TCP || b.DOT {
		network = "tcp"
	}

	var query queryFunc
	if b.useDoH {
		query, network = b.getDoHClient()
	}

	if b.useQuic {
		// nolint:gosec
		quicClient, err := doq.NewClient(b.Server, doq.Options{TlsConfig: &tls.Config{InsecureSkipVerify: b.Insecure}})
		if err != nil {
			return nil, err
		}
		query = func(ctx context.Context, _ string, msg *dns.Msg) (*dns.Msg, error) {
			return quicClient.Send(ctx, msg)
		}
		network = "quic"
	}

	limits := ""
	var limit ratelimit.Limiter
	if b.Rate > 0 {
		limit = ratelimit.New(b.Rate)
		limits = fmt.Sprintf("(limited to %s QPS)", highlightStr(b.Rate))
	}

	if !b.Silent && !b.JSON {
		fmt.Printf("Benchmarking %s via %s with %s concurrent requests %s\n", highlightStr(b.Server), highlightStr(network), highlightStr(b.Concurrency), limits)
	}

	stats := make([]*ResultStats, b.Concurrency)

	var wg sync.WaitGroup
	var w uint32
	for w = 0; w < b.Concurrency; w++ {
		st := &ResultStats{Hist: hdrhistogram.New(b.HistMin.Nanoseconds(), b.HistMax.Nanoseconds(), b.HistPre)}
		stats[w] = st
		if b.Rcodes {
			st.Codes = make(map[int]int64)
		}
		st.Qtypes = make(map[string]int64)
		st.Counters = &Counters{}

		var co *dns.Conn
		var err error
		wg.Add(1)
		go func(st *ResultStats) {
			defer func() {
				if co != nil {
					co.Close()
				}
				wg.Done()
			}()

			// create a new lock free rand source for this goroutine
			// nolint:gosec
			rando := rand.New(rand.NewSource(time.Now().Unix()))

			var i int64
			for i = 0; i < b.Count || b.Duration != 0; i++ {
				for _, qt := range qTypes {
					for _, q := range questions {
						if rando.Float64() > b.Probability {
							continue
						}
						var r *dns.Msg
						m := dns.Msg{}
						m.RecursionDesired = b.Recurse
						m.Question = make([]dns.Question, 1)
						question := dns.Question{Qtype: qt, Qclass: dns.ClassINET}
						if ctx.Err() != nil {
							return
						}
						st.Counters.Total++

						// instead of setting the question, do this manually for lower overhead and lock free access to id
						question.Name = q
						if b.useQuic {
							m.Id = 0
						} else {
							m.Id = uint16(rando.Uint32())
						}
						m.Question[0] = question
						if limit != nil {
							limit.Take()
						}

						start := time.Now()
						if b.useQuic || b.useDoH {
							r, err = query(ctx, b.Server, &m)
							if err != nil {
								st.Counters.IOError++
								st.Errors = append(st.Errors, err)
								continue
							}
						} else {
							if co != nil && b.QperConn > 0 && i%b.QperConn == 0 {
								co.Close()
								co = nil
							}

							if co == nil {
								co, err = dialConnection(b, &m, st)
								if err != nil {
									st.Errors = append(st.Errors, err)
									continue
								}
							}

							co.SetWriteDeadline(start.Add(b.WriteTimeout))
							if err = co.WriteMsg(&m); err != nil {
								// error writing
								st.Errors = append(st.Errors, err)
								st.Counters.IOError++
								co.Close()
								co = nil
								continue
							}

							co.SetReadDeadline(time.Now().Add(b.ReadTimeout))

							r, err = co.ReadMsg()
							if err != nil {
								// error reading
								st.Errors = append(st.Errors, err)
								st.Counters.IOError++
								co.Close()
								co = nil
								continue
							}
						}

						st.record(&m, r, start, time.Since(start))
					}
				}
			}
		}(st)
	}

	wg.Wait()

	return stats, nil
}

func (b *Benchmark) addPortIfMissing() {
	if b.useDoH {
		// both HTTPS and HTTP are using default ports 443 and 80 if no other port is specified
		return
	}
	if _, _, err := net.SplitHostPort(b.Server); err != nil {
		if b.DOT {
			// https://www.rfc-editor.org/rfc/rfc7858
			b.Server = net.JoinHostPort(b.Server, "853")
			return
		}
		if b.useQuic {
			b.Server = net.JoinHostPort(b.Server, "853")
			return
		}
		b.Server = net.JoinHostPort(b.Server, "53")
		return
	}
	if ip := net.ParseIP(b.Server); ip != nil {
		b.Server = net.JoinHostPort(ip.String(), "53")
		return
	}
}

func isHTTPUrl(s string) (ok bool, network string) {
	if strings.HasPrefix(s, "http://") {
		return true, "http"
	}
	if strings.HasPrefix(s, "https://") {
		return true, "https"
	}
	return false, ""
}

func (b *Benchmark) getDoHClient() (queryFunc, string) {
	_, network := isHTTPUrl(b.Server)
	var tr http.RoundTripper
	switch b.DohProtocol {
	case "3":
		network += "/3"
		// nolint:gosec
		tr = &http3.RoundTripper{TLSClientConfig: &tls.Config{InsecureSkipVerify: b.Insecure}}
	case "2":
		network += "/2"
		// nolint:gosec
		tr = &http2.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: b.Insecure}}
	case "1.1":
		fallthrough
	default:
		network += "/1.1"
		// nolint:gosec
		tr = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: b.Insecure}}
	}
	c := http.Client{Transport: tr, Timeout: b.ReadTimeout}
	dohClient := doh.NewClient(&c)

	switch b.DohMethod {
	case "post":
		network += " (POST)"
		return dohClient.SendViaPost, network
	case "get":
		network += " (GET)"
		return dohClient.SendViaGet, network
	default:
		network += " (POST)"
		return dohClient.SendViaPost, network
	}
}
