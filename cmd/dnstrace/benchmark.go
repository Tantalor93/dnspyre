package dnstrace

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/miekg/dns"
	"github.com/tantalor93/doh-go/doh"
	"go.uber.org/ratelimit"
	"golang.org/x/net/http2"
)

const dnsTimeout = time.Second * 4

type rstats struct {
	codes   map[int]int64
	qtypes  map[string]int64
	hist    *hdrhistogram.Histogram
	timings []datapoint
}

type datapoint struct {
	duration float64
	start    time.Time
}

func (r *rstats) record(time time.Time, timing time.Duration) {
	r.hist.RecordValue(timing.Nanoseconds())
	r.timings = append(r.timings, datapoint{float64(timing.Milliseconds()), time})
}

func do(ctx context.Context) []*rstats {
	questions := make([]string, len(*pQueries))
	for i, q := range *pQueries {
		questions[i] = dns.Fqdn(q)
	}

	fmt.Printf("Using %d hostnames\n", len(questions))

	var qTypes []uint16
	for _, v := range *pTypes {
		qTypes = append(qTypes, dns.StringToType[v])
	}

	srv := *pServer

	useDoH := strings.HasPrefix(*pServer, "http")

	if !strings.Contains(srv, ":") && !useDoH {
		srv += ":53"
	}

	network := "udp"
	if *pTCP || *pDOT {
		network = "tcp"
	}

	var dohClient doh.Client
	var dohFunc func(context.Context, string, *dns.Msg) (*dns.Msg, error)
	if useDoH {
		network = "https"
		var tr http.RoundTripper
		switch *pDoHProtocol {
		case "1.1":
			network = network + "/1.1"
			tr = &http.Transport{}
		case "2":
			network = network + "/2"
			tr = &http2.Transport{}
		default:
			network = network + "/1.1"
			tr = &http.Transport{}
		}
		c := http.Client{Transport: tr, Timeout: *pReadTimeout}
		dohClient = *doh.NewClient(&c)

		switch *pDoHmethod {
		case "post":
			network = network + " (POST)"
			dohFunc = dohClient.SendViaPost
		case "get":
			network = network + " (GET)"
			dohFunc = dohClient.SendViaGet
		default:
			network = network + " (POST)"
			dohFunc = dohClient.SendViaPost
		}
	}

	concurrent := *pConcurrency

	limits := ""
	var limit ratelimit.Limiter
	if *pRate > 0 {
		limit = ratelimit.New(*pRate)
		limits = fmt.Sprintf("(limited to %d QPS)", *pRate)
	}

	if !*pSilent {
		fmt.Printf("Benchmarking %s via %s with %d concurrent requests %s\n", srv, network, concurrent, limits)
	}

	stats := make([]*rstats, concurrent)

	var wg sync.WaitGroup
	var w uint32
	for w = 0; w < concurrent; w++ {
		st := &rstats{hist: hdrhistogram.New(pHistMin.Nanoseconds(), pHistMax.Nanoseconds(), *pHistPre)}
		stats[w] = st
		if *pRCodes {
			st.codes = make(map[int]int64)
		}
		st.qtypes = make(map[string]int64)

		var co *dns.Conn
		var err error
		wg.Add(1)
		go func(st *rstats) {
			defer func() {
				if co != nil {
					co.Close()
				}
				wg.Done()
			}()

			// create a new lock free rand source for this goroutine
			rando := rand.New(rand.NewSource(time.Now().Unix()))

			var i int64
			for i = 0; i < *pCount; i++ {
				for _, qt := range qTypes {
					for _, q := range questions {
						if rando.Float64() > *pProbability {
							continue
						}
						var r *dns.Msg
						m := dns.Msg{}
						m.RecursionDesired = *pRecurse
						m.Question = make([]dns.Question, 1)
						question := dns.Question{Qtype: qt, Qclass: dns.ClassINET}
						if ctx.Err() != nil {
							return
						}
						atomic.AddInt64(&count, 1)

						// instead of setting the question, do this manually for lower overhead and lock free access to id
						question.Name = q
						m.Id = uint16(rando.Uint32())
						m.Question[0] = question
						if limit != nil {
							limit.Take()
						}

						start := time.Now()
						if useDoH {
							r, err = dohFunc(ctx, *pServer, &m)
							if err != nil {
								atomic.AddInt64(&ecount, 1)
								continue
							}
						} else {
							if co != nil && *pQperConn > 0 && i%*pQperConn == 0 {
								co.Close()
								co = nil
							}

							if co == nil {
								co, err = dialConnection(srv, network, &m)
								if err != nil {
									continue
								}
							}

							co.SetWriteDeadline(start.Add(*pWriteTimeout))
							if err = co.WriteMsg(&m); err != nil {
								// error writing
								atomic.AddInt64(&ecount, 1)
								if *pIOErrors {
									fmt.Fprintln(os.Stderr, "i/o error dialing: ", err)
								}
								co.Close()
								co = nil
								continue
							}

							co.SetReadDeadline(time.Now().Add(*pReadTimeout))

							r, err = co.ReadMsg()
							if err != nil {
								// error reading
								atomic.AddInt64(&ecount, 1)
								if *pIOErrors {
									fmt.Fprintln(os.Stderr, "i/o error dialing: ", err)
								}
								co.Close()
								co = nil
								continue
							}
						}

						st.record(start, time.Since(start))
						evaluateResponse(r, &m, st)
					}
				}
			}
		}(st)
	}

	wg.Wait()

	return stats
}

func evaluateResponse(r *dns.Msg, q *dns.Msg, st *rstats) {
	if r.Truncated {
		atomic.AddInt64(&truncated, 1)
	}

	if r.Rcode == dns.RcodeSuccess {
		if r.Id != q.Id {
			atomic.AddInt64(&mismatch, 1)
			return
		}
		atomic.AddInt64(&success, 1)

		if expect := *pExpect; len(expect) > 0 {
			for _, s := range r.Answer {
				a := dns.TypeToString[s.Header().Rrtype]
				ok := isExpected(a)

				if ok {
					atomic.AddInt64(&matched, 1)
					break
				}
			}
		}
	}

	if st.codes != nil {
		var c int64
		if v, ok := st.codes[r.Rcode]; ok {
			c = v
		}
		c++
		st.codes[r.Rcode] = c
	}
	if st.qtypes != nil {
		st.qtypes[dns.TypeToString[q.Question[0].Qtype]]++
	}
}

func isExpected(a string) bool {
	for _, b := range *pExpect {
		if b == a {
			return true
		}
	}
	return false
}
