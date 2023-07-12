package cmd

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strconv"
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

var client = http.Client{
	Timeout: 120 * time.Second,
}

// Benchmark is representation of benchmark scenario.
type Benchmark struct {
	Server      string
	Types       []string
	Count       int64
	Concurrency uint32

	Rate            int
	RateLimitWorker int
	QperConn        int64

	Recurse bool

	Probability float64

	UDPSize uint16
	EdnsOpt string

	TCP bool
	DOT bool

	WriteTimeout   time.Duration
	ReadTimeout    time.Duration
	ConnectTimeout time.Duration
	RequestTimeout time.Duration

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

	if b.HistMax == 0 {
		b.HistMax = b.RequestTimeout
	}
	return nil
}

// Run executes benchmark, if benchmark is unable to start the error is returned, otherwise array of results from parallel benchmark goroutines is returned.
func (b *Benchmark) Run(ctx context.Context) ([]*ResultStats, error) {
	if err := b.normalize(); err != nil {
		return nil, err
	}

	color.NoColor = !b.Color

	questions, err := b.prepareQuestions()
	if err != nil {
		return nil, err
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
	if b.TCP {
		network = "tcp"
	}
	if b.DOT {
		network = "tls"
	}

	var query queryFunc
	if b.useDoH {
		var dohQuery queryFunc
		dohQuery, network = b.getDoHClient()
		query = func(ctx context.Context, s string, msg *dns.Msg) (*dns.Msg, error) {
			return dohQuery(ctx, s, msg)
		}
	}

	if b.useQuic {
		h, _, _ := net.SplitHostPort(b.Server)
		// nolint:gosec
		quicClient := doq.NewClient(b.Server, doq.Options{
			TLSConfig:      &tls.Config{ServerName: h, InsecureSkipVerify: b.Insecure},
			ReadTimeout:    b.ReadTimeout,
			WriteTimeout:   b.WriteTimeout,
			ConnectTimeout: b.ConnectTimeout,
		})
		query = func(ctx context.Context, _ string, msg *dns.Msg) (*dns.Msg, error) {
			return quicClient.Send(ctx, msg)
		}
		network = "quic"
	}

	limits := ""
	var limit ratelimit.Limiter
	if b.Rate > 0 {
		limit = ratelimit.New(b.Rate)
		if b.RateLimitWorker == 0 {
			limits = fmt.Sprintf("(limited to %s QPS overall)", highlightStr(b.Rate))
		} else {
			limits = fmt.Sprintf("(limited to %s QPS overall and %s QPS per concurrent worker)", highlightStr(b.Rate), highlightStr(b.RateLimitWorker))
		}
	}
	if b.Rate == 0 && b.RateLimitWorker > 0 {
		limits = fmt.Sprintf("(limited to %s QPS per concurrent worker)", highlightStr(b.RateLimitWorker))
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

		var err error
		wg.Add(1)
		go func(st *ResultStats) {
			defer func() {
				wg.Done()
			}()

			// create a new lock free rand source for this goroutine
			// nolint:gosec
			rando := rand.New(rand.NewSource(time.Now().Unix()))

			var workerLimit ratelimit.Limiter
			if b.RateLimitWorker > 0 {
				workerLimit = ratelimit.New(b.RateLimitWorker)
			}

			var i int64

			// shadow & copy the query func, because for DoQ and DoH we want to share the client, for plain DNS and DoT we don't
			// due to manual connection redialing on error, etc.
			query := query

			if query == nil {
				dnsClient := b.getDNSClient()

				var co *dns.Conn
				query = func(ctx context.Context, s string, msg *dns.Msg) (*dns.Msg, error) {
					if co != nil && b.QperConn > 0 && i%b.QperConn == 0 {
						co.Close()
						co = nil
					}

					if co == nil {
						co, err = dnsClient.Dial(b.Server)
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

			for i = 0; i < b.Count || b.Duration != 0; i++ {
				for _, qt := range qTypes {
					for _, q := range questions {
						if ctx.Err() != nil {
							return
						}
						if rando.Float64() > b.Probability {
							continue
						}
						if limit != nil {
							if err := checkLimit(ctx, limit); err != nil {
								return
							}
						}
						if workerLimit != nil {
							if err := checkLimit(ctx, workerLimit); err != nil {
								return
							}
						}
						var resp *dns.Msg

						m := dns.Msg{}
						m.RecursionDesired = b.Recurse

						m.Question = make([]dns.Question, 1)
						question := dns.Question{Name: q, Qtype: qt, Qclass: dns.ClassINET}
						m.Question[0] = question

						if b.useQuic {
							m.Id = 0
						} else {
							m.Id = uint16(rando.Uint32())
						}

						if ednsOpt := b.EdnsOpt; len(ednsOpt) > 0 {
							addEdnsOpt(&m, ednsOpt)
						}

						st.Counters.Total++

						start := time.Now()

						reqTimeoutCtx, cancel := context.WithTimeout(ctx, b.RequestTimeout)
						if resp, err = query(reqTimeoutCtx, b.Server, &m); err != nil {
							cancel()
							st.Counters.IOError++
							st.Errors = append(st.Errors, err)
							continue
						}

						cancel()
						st.record(&m, resp, start, time.Since(start))
					}
				}
			}
		}(st)
	}

	wg.Wait()

	return stats, nil
}

func addEdnsOpt(m *dns.Msg, ednsOpt string) {
	o := m.IsEdns0()
	if o == nil {
		m.SetEdns0(4096, true)
		o = m.IsEdns0()
	}
	s := strings.Split(ednsOpt, ":")
	data, err := hex.DecodeString(s[1])
	if err != nil {
		panic(err)
	}
	code, err := strconv.ParseUint(s[0], 10, 16)
	if err != nil {
		panic(err)
	}
	o.Option = append(o.Option, &dns.EDNS0_LOCAL{Code: uint16(code), Data: data})
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
			// https://datatracker.ietf.org/doc/rfc9250
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

func (b *Benchmark) getDNSClient() *dns.Client {
	network := "udp"
	if b.TCP {
		network = "tcp"
	} else if b.DOT {
		network = "tcp-tls"
	}

	dnsClient := dns.Client{
		Net:          network,
		DialTimeout:  b.ConnectTimeout,
		WriteTimeout: b.WriteTimeout,
		ReadTimeout:  b.ReadTimeout,
		Timeout:      b.RequestTimeout,
	}
	return &dnsClient
}

func (b *Benchmark) prepareQuestions() ([]string, error) {
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
	return questions, nil
}

func checkLimit(ctx context.Context, limiter ratelimit.Limiter) error {
	done := make(chan struct{})
	go func() {
		limiter.Take()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
