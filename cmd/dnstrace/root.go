package dnstrace

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/codahale/hdrhistogram"
	"github.com/fatih/color"
	"github.com/miekg/dns"
	"github.com/tantalor93/dnstrace/internal/sysutil"
	"go.uber.org/ratelimit"
)

var (
	// Version is set during release of project during build process
	Version = "development"
	author  = "Ondrej Benkovsky <obenky@gmail.com>, Rahul Powar <rahul@redsift.io>"

	logger    = log.New(os.Stdout, "", 0)
	errLogger = log.New(os.Stderr, "", 0)
)

var (
	pApp = kingpin.New("dnstrace", "A high QPS DNS benchmark.").Author(author)

	pServer = pApp.Flag("server", "DNS server IP:port to test. IPv6 is also supported, for example '[fddd:dddd::]:53'.").Short('s').Default("127.0.0.1").String()
	pType   = pApp.Flag("type", "Query type.").Short('t').Default("A").Enum(getSupportedDNSTypes()...)

	pCount       = pApp.Flag("number", "Number of queries to issue. Note that the total number of queries issued = number*concurrency*len(queries).").Short('n').Default("1").Int64()
	pConcurrency = pApp.Flag("concurrency", "Number of concurrent queries to issue.").Short('c').Default("1").Uint32()
	pRate        = pApp.Flag("rate-limit", "Apply a global questions / second rate limit.").Short('l').Default("0").Int()
	pQperConn    = pApp.Flag("query-per-conn", "Queries on a connection before creating a new one. 0: unlimited").Default("0").Int64()

	pExpect = pApp.Flag("expect", "Expect a specific response.").Short('e').Strings()

	pRecurse     = pApp.Flag("recurse", "Allow DNS recursion.").Short('r').Default("false").Bool()
	pProbability = pApp.Flag("probability", "Each hostname from file will be used with provided probability in %. Value 1 and above means that each hostname from file will be used by each concurrent benchmark goroutine. Useful for randomizing queries across benchmark goroutines.").Default("1").Float64()
	pUDPSize     = pApp.Flag("edns0", "Enable EDNS0 with specified size.").Default("0").Uint16()
	pEdnsOpt     = pApp.Flag("ednsopt", "code[:value], Specify EDNS option with code point code and optionally payload of value as a hexadecimal string. code must be arbitrary numeric value.").Default("").String()
	pTCP         = pApp.Flag("tcp", "Use TCP fot DNS requests.").Default("false").Bool()
	pDOT         = pApp.Flag("dot", "Use DoT for DNS requests.").Default("false").Bool()

	pWriteTimeout = pApp.Flag("write", "DNS write timeout.").Default("1s").Duration()
	pReadTimeout  = pApp.Flag("read", "DNS read timeout.").Default(dnsTimeout.String()).Duration()

	pRCodes = pApp.Flag("codes", "Enable counting DNS return codes.").Default("true").Bool()

	pHistMin     = pApp.Flag("min", "Minimum value for timing histogram.").Default((time.Microsecond * 400).String()).Duration()
	pHistMax     = pApp.Flag("max", "Maximum value for histogram.").Default(dnsTimeout.String()).Duration()
	pHistPre     = pApp.Flag("precision", "Significant figure for histogram precision.").Default("1").PlaceHolder("[1-5]").Int()
	pHistDisplay = pApp.Flag("distribution", "Display distribution histogram of timings to stdout.").Default("true").Bool()
	pCsv         = pApp.Flag("csv", "Export distribution to CSV.").Default("").PlaceHolder("/path/to/file.csv").String()

	pIOErrors = pApp.Flag("io-errors", "Log I/O errors to stderr.").Default("false").Bool()

	pSilent = pApp.Flag("silent", "Disable stdout.").Default("false").Bool()
	pColor  = pApp.Flag("color", "ANSI Color output.").Default("true").Bool()

	pQueries = pApp.Arg("queries", "Queries to issue. Can be file referenced using @<file-path>, for example @data/2-domains").Required().Strings()
)

var (
	count     int64
	cerror    int64
	ecount    int64
	success   int64
	matched   int64
	mismatch  int64
	truncated int64
)

const (
	dnsTimeout   = time.Second * 4
	fileNoBuffer = 9 // app itself needs about 9 for libs
)

type rstats struct {
	codes map[int]int64
	hist  *hdrhistogram.Histogram
}

func isExpected(a string) bool {
	for _, b := range *pExpect {
		if b == a {
			return true
		}
	}
	return false
}

func do(ctx context.Context) []*rstats {
	questions := make([]string, len(*pQueries))
	for i, q := range *pQueries {
		questions[i] = dns.Fqdn(q)
	}

	logger.Printf("Using %d hostnames", len(questions))

	qType := dns.StringToType[*pType]

	srv := *pServer
	if !strings.Contains(srv, ":") {
		srv += ":53"
	}

	network := "udp"
	if *pTCP || *pDOT {
		network = "tcp"
	}

	concurrent := *pConcurrency

	limits := ""
	var limit ratelimit.Limiter
	if *pRate > 0 {
		limit = ratelimit.New(*pRate)
		limits = fmt.Sprintf("(limited to %d QPS)", *pRate)
	}

	if !*pSilent {
		logger.Printf("Benchmarking %s via %s with %d concurrent requests %s", srv, network, concurrent, limits)
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

			var r *dns.Msg
			m := new(dns.Msg)
			m.RecursionDesired = *pRecurse
			m.Question = make([]dns.Question, 1)
			question := dns.Question{Qtype: qType, Qclass: dns.ClassINET}

			// create a new lock free rand source for this goroutine
			rando := rand.New(rand.NewSource(time.Now().Unix()))

			var i int64
			for i = 0; i < *pCount; i++ {
				for _, q := range questions {
					if rand.Float64() > *pProbability {
						continue
					}
					if ctx.Err() != nil {
						return
					}
					if co != nil && *pQperConn > 0 && i%*pQperConn == 0 {
						co.Close()
						co = nil
					}
					atomic.AddInt64(&count, 1)

					// instead of setting the question, do this manually for lower overhead and lock free access to id
					question.Name = q
					m.Id = uint16(rando.Uint32())
					m.Question[0] = question

					if co == nil {
						co, err = dial(srv, network)
						if err != nil {
							atomic.AddInt64(&cerror, 1)

							if *pIOErrors {
								errLogger.Println("i/o error dialing: ", err.Error())
							}
							continue
						}
						if udpSize := *pUDPSize; udpSize > 0 {
							m.SetEdns0(udpSize, true)
							co.UDPSize = udpSize
						}
						if ednsOpt := *pEdnsOpt; len(ednsOpt) > 0 {
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
					}

					if limit != nil {
						limit.Take()
					}

					start := time.Now()
					co.SetWriteDeadline(start.Add(*pWriteTimeout))
					if err = co.WriteMsg(m); err != nil {
						// error writing
						atomic.AddInt64(&ecount, 1)
						if *pIOErrors {
							errLogger.Println("i/o error dialing: ", err.Error())
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
							errLogger.Println("i/o error dialing: ", err.Error())
						}
						co.Close()
						co = nil
						continue
					}
					timing := time.Since(start)

					st.hist.RecordValue(timing.Nanoseconds())

					if r.Truncated {
						atomic.AddInt64(&truncated, 1)
					}

					if r.Rcode == dns.RcodeSuccess {
						if r.Id != m.Id {
							atomic.AddInt64(&mismatch, 1)
							continue
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
				}
			}
		}(st)
	}

	wg.Wait()

	return stats
}

func dial(srv string, network string) (*dns.Conn, error) {
	if *pDOT {
		return dns.DialTimeoutWithTLS(network, srv, &tls.Config{}, dnsTimeout)
	}
	return dns.DialTimeout(network, srv, dnsTimeout)
}

// Execute starts main logic of command
func Execute() {
	pApp.Version(Version)
	kingpin.MustParse(pApp.Parse(os.Args[1:]))

	// process args
	color.NoColor = !*pColor

	lim, err := sysutil.RlimitStack()
	if err != nil {
		logger.Println("Cannot check limit of number of files. Skipping check. Please make sure it is sufficient manually.", err)
	} else {
		needed := uint64(*pConcurrency) + uint64(fileNoBuffer)
		if lim < needed {
			logger.Fatalf("Current process limit for number of files is %d and insufficient for level of requested concurrency.", lim)
		}
	}

	var csv *os.File
	if *pCsv != "" {
		f, err := os.Create(*pCsv)
		if err != nil {
			logger.Fatalln("Failed to create file for CSV export.")
		}

		csv = f
	}

	sigsInt := make(chan os.Signal, 8)
	signal.Notify(sigsInt, syscall.SIGINT)

	sigsHup := make(chan os.Signal, 8)
	signal.Notify(sigsHup, syscall.SIGHUP)

	defer close(sigsInt)
	defer close(sigsHup)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-sigsInt
		printProgress()
		errLogger.Println("Cancelling benchmark ^C, again to terminate now.")
		cancel()
		<-sigsInt
		os.Exit(130)
	}()
	go func() {
		for range sigsHup {
			printProgress()
		}
	}()

	// get going
	rand.Seed(time.Now().UnixNano())

	start := time.Now()
	res := do(ctx)
	end := time.Now()

	printReport(end.Sub(start), res, csv)

	if cerror > 0 || ecount > 0 || mismatch > 0 {
		// something was wrong
		os.Exit(1)
	}
}

func getSupportedDNSTypes() []string {
	keys := make([]string, 0, len(dns.StringToType))
	for k := range dns.StringToType {
		keys = append(keys, k)
	}
	return keys
}
