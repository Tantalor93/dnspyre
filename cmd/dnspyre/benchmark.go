package dnspyre

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/fatih/color"
	"github.com/miekg/dns"
	"github.com/tantalor93/doh-go/doh"
	"go.uber.org/ratelimit"
	"golang.org/x/net/http2"
)

const dnsTimeout = time.Second * 4

var client = http.Client{
	Timeout: 120 * time.Second,
}

// ResultStats is a representation of benchmark results of single concurrent thread.
type ResultStats struct {
	Codes    map[int]int64
	Qtypes   map[string]int64
	Hist     *hdrhistogram.Histogram
	Timings  []Datapoint
	Counters *Counters
}

// Counters represents various counters of benchmark results.
type Counters struct {
	Total      int64
	ConnError  int64
	IOError    int64
	Success    int64
	IDmismatch int64
	Truncated  int64
}

func (r *ResultStats) record(time time.Time, timing time.Duration) {
	r.Hist.RecordValue(timing.Nanoseconds())
	r.Timings = append(r.Timings, Datapoint{float64(timing.Milliseconds()), time})
}

// Datapoint one datapoint of benchmark (single DNS request).
type Datapoint struct {
	Duration float64
	Start    time.Time
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

	Csv string

	Silent bool
	Color  bool

	PlotDir    string
	PlotFormat string

	DohMethod   string
	DohProtocol string

	Queries []string

	Duration time.Duration

	// internal variable so we do not have to parse the address with each request.
	useDoH bool
}

func (b *Benchmark) normalize() {
	b.useDoH = strings.HasPrefix(b.Server, "http")

	if !strings.Contains(b.Server, ":") && !b.useDoH {
		b.Server += ":53"
	}

	if b.Count == 0 && b.Duration == 0 {
		b.Count = 1
	}

	if b.Duration > 0 && b.Count > 0 {
		fmt.Fprintln(os.Stderr, "--number and --duration is specified at once, only one can be used")
		os.Exit(1)
	}
}

// Run executes benchmark.
func (b *Benchmark) Run(ctx context.Context) []*ResultStats {
	b.normalize()

	color.NoColor = !b.Color

	var questions []string
	for _, q := range b.Queries {
		if strings.HasPrefix(q, "http://") || strings.HasPrefix(q, "https://") {
			resp, err := client.Get(q)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to download file '%s' with error '%v'", q, err)
				os.Exit(1)
			}
			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				fmt.Fprintf(os.Stderr, "Failed to download file '%s' with status '%s'", q, resp.Status)
				os.Exit(1)
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

	if !b.Silent {
		fmt.Printf("Using %d hostnames\n", len(questions))
	}

	var qTypes []uint16
	for _, v := range b.Types {
		qTypes = append(qTypes, dns.StringToType[v])
	}

	network := "udp"
	if b.TCP || b.DOT {
		network = "tcp"
	}

	var dohClient doh.Client
	var dohFunc func(context.Context, string, *dns.Msg) (*dns.Msg, error)
	if b.useDoH {
		network = "https"
		var tr http.RoundTripper
		switch b.DohProtocol {
		case "1.1":
			network += "/1.1"
			tr = &http.Transport{}
		case "2":
			network += "/2"
			tr = &http2.Transport{}
		default:
			network += "/1.1"
			tr = &http.Transport{}
		}
		c := http.Client{Transport: tr, Timeout: b.ReadTimeout}
		dohClient = *doh.NewClient(&c)

		switch b.DohMethod {
		case "post":
			network += " (POST)"
			dohFunc = dohClient.SendViaPost
		case "get":
			network += " (GET)"
			dohFunc = dohClient.SendViaGet
		default:
			network += " (POST)"
			dohFunc = dohClient.SendViaPost
		}
	}

	limits := ""
	var limit ratelimit.Limiter
	if b.Rate > 0 {
		limit = ratelimit.New(b.Rate)
		limits = fmt.Sprintf("(limited to %d QPS)", b.Rate)
	}

	if !b.Silent {
		fmt.Printf("Benchmarking %s via %s with %d concurrent requests %s\n", b.Server, network, b.Concurrency, limits)
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
						m.Id = uint16(rando.Uint32())
						m.Question[0] = question
						if limit != nil {
							limit.Take()
						}

						start := time.Now()
						if b.useDoH {
							r, err = dohFunc(ctx, b.Server, &m)
							if err != nil {
								st.Counters.IOError++
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
									continue
								}
							}

							co.SetWriteDeadline(start.Add(b.WriteTimeout))
							if err = co.WriteMsg(&m); err != nil {
								// error writing
								st.Counters.IOError++
								co.Close()
								co = nil
								continue
							}

							co.SetReadDeadline(time.Now().Add(b.ReadTimeout))

							r, err = co.ReadMsg()
							if err != nil {
								// error reading
								st.Counters.IOError++
								co.Close()
								co = nil
								continue
							}
						}

						st.record(start, time.Since(start))
						b.evaluateResponse(r, &m, st)
					}
				}
			}
		}(st)
	}

	wg.Wait()

	return stats
}

func (b *Benchmark) evaluateResponse(r *dns.Msg, q *dns.Msg, st *ResultStats) {
	if r.Truncated {
		st.Counters.Truncated++
	}

	if r.Rcode == dns.RcodeSuccess {
		if r.Id != q.Id {
			st.Counters.IDmismatch++
			return
		}
		st.Counters.Success++
	}

	if st.Codes != nil {
		var c int64
		if v, ok := st.Codes[r.Rcode]; ok {
			c = v
		}
		c++
		st.Codes[r.Rcode] = c
	}
	if st.Qtypes != nil {
		st.Qtypes[dns.TypeToString[q.Question[0].Qtype]]++
	}
}
