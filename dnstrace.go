package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/alecthomas/kingpin"
	"github.com/codahale/hdrhistogram"
	"github.com/fatih/color"
	"github.com/miekg/dns"
	"github.com/olekukonko/tablewriter"
	"go.uber.org/ratelimit"
)

var (
	// Tag is set by build at compile time to Git Tag
	Tag = ""
	// Commit is set by build at compile time to Git SHA1
	Commit = ""
	author = "Rahul Powar <rahul@redsift.io>"
)

var (
	pApp = kingpin.New("dnstrace", "A high QPS DNS benchmark.").Author(author)

	pServer = pApp.Flag("server", "DNS server IP:port to test.").Short('s').Default("127.0.0.1").String()
	pType   = pApp.Flag("type", "Query type.").Short('t').Default("A").Enum(getSupportedDNSTypes()...)

	pCount       = pApp.Flag("number", "Number of queries to issue. Note that the total number of queries issued = number*concurrency*len(queries).").Short('n').Default("1").Int64()
	pConcurrency = pApp.Flag("concurrency", "Number of concurrent queries to issue.").Short('c').Default("1").Uint32()
	pRate        = pApp.Flag("rate-limit", "Apply a global questions / second rate limit.").Short('l').Default("0").Int()
	pQperConn    = pApp.Flag("query-per-conn", "Queries on a connection before creating a new one. 0: unlimited").Default("0").Int64()

	pExpect = pApp.Flag("expect", "Expect a specific response.").Short('e').Strings()

	pRecurse     = pApp.Flag("recurse", "Allow DNS recursion.").Short('r').Default("false").Bool()
	pProbability = pApp.Flag("probability", "Each hostname from file will be used with provided probability in %. Value 1 and above means that each hostname from file will be used by each concurrent benchmark goroutine. Useful for randomizing queries accross benchmark goroutines.").Default("1").Float64()
	pUDPSize     = pApp.Flag("edns0", "Enable EDNS0 with specified size.").Default("0").Uint16()
	pTCP         = pApp.Flag("tcp", "Use TCP fot DNS requests.").Default("false").Bool()

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

const dnsTimeout = time.Second * 4

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

	fmt.Printf("Using %d hostnames\n\n", len(questions))

	qType := dns.StringToType[*pType]

	srv := *pServer
	if !strings.Contains(srv, ":") {
		srv += ":53"
	}

	network := "udp"
	if *pTCP {
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
		fmt.Printf("Benchmarking %s via %s with %d concurrent requests %s\n\n", srv, network, concurrent, limits)
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
						co, err = dns.DialTimeout(network, srv, dnsTimeout)
						if err != nil {
							atomic.AddInt64(&cerror, 1)

							if *pIOErrors {
								fmt.Fprintln(os.Stderr, "i/o error dialing: ", err.Error())
							}
							continue
						}
						if udpSize := *pUDPSize; udpSize > 0 {
							m.SetEdns0(udpSize, true)
							co.UDPSize = udpSize
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
							fmt.Fprintln(os.Stderr, "i/o error writing: ", err.Error())
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
							fmt.Fprintln(os.Stderr, "i/o error reading: ", err.Error())
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

func printProgress() {
	if *pSilent {
		return
	}

	fmt.Println()

	errorFprint := color.New(color.FgRed).Fprint
	successFprint := color.New(color.FgGreen).Fprint

	acount := atomic.LoadInt64(&count)
	acerror := atomic.LoadInt64(&cerror)
	aecount := atomic.LoadInt64(&ecount)
	amismatch := atomic.LoadInt64(&mismatch)
	asuccess := atomic.LoadInt64(&success)
	amatched := atomic.LoadInt64(&matched)
	atruncated := atomic.LoadInt64(&truncated)

	fmt.Printf("Total requests:\t %d\t\n", acount)

	if acerror > 0 || aecount > 0 {
		errorFprint(os.Stdout, "Connection errors:\t", acerror, "\n")
		errorFprint(os.Stdout, "Read/Write errors:\t", aecount, "\n")
	}

	if amismatch > 0 {
		errorFprint(os.Stdout, "ID mismatch errors:\t", amismatch, "\n")
	}

	successFprint(os.Stdout, "DNS success codes:\t", asuccess, "\n")

	if atruncated > 0 {
		errorFprint(os.Stdout, "Truncated responses:\t", atruncated, "\n")
	} else {
		successFprint(os.Stdout, "Truncated responses:\t", atruncated, "\n")
	}

	if len(*pExpect) > 0 {
		expect := successFprint
		if amatched != asuccess {
			expect = errorFprint
		}
		expect(os.Stdout, "Expected results:\t", amatched, "\n")
	}
}

func printReport(t time.Duration, stats []*rstats, csv *os.File) {
	defer func() {
		if csv != nil {
			csv.Close()
		}
	}()

	// merge all the stats here
	timings := hdrhistogram.New(pHistMin.Nanoseconds(), pHistMax.Nanoseconds(), *pHistPre)
	codeTotals := make(map[int]int64)
	for _, s := range stats {
		timings.Merge(s.hist)
		if s.codes != nil {
			for k, v := range s.codes {
				codeTotals[k] = codeTotals[k] + v
			}
		}
	}

	if csv != nil {
		writeBars(csv, timings.Distribution())

		fmt.Println()
		fmt.Println("DNS distribution written to", csv.Name())
	}

	if *pSilent {
		return
	}

	printProgress()

	if len(codeTotals) > 0 {
		errorFprint := color.New(color.FgRed).Fprint
		successFprint := color.New(color.FgGreen).Fprint

		fmt.Println()
		fmt.Println("DNS response codes")
		for i := dns.RcodeSuccess; i <= dns.RcodeBadCookie; i++ {
			printFn := errorFprint
			if i == dns.RcodeSuccess {
				printFn = successFprint
			}
			if c, ok := codeTotals[i]; ok {
				printFn(os.Stdout, "\t", dns.RcodeToString[i]+":\t", c, "\n")
			}
		}
	}

	fmt.Println()

	fmt.Println("Time taken for tests:\t", t.String())
	fmt.Printf("Questions per second:\t %0.1f\n", float64(count)/t.Seconds())

	min := time.Duration(timings.Min())
	mean := time.Duration(timings.Mean())
	sd := time.Duration(timings.StdDev())
	max := time.Duration(timings.Max())
	p99 := time.Duration(timings.ValueAtQuantile(99))
	p95 := time.Duration(timings.ValueAtQuantile(95))
	p90 := time.Duration(timings.ValueAtQuantile(90))
	p75 := time.Duration(timings.ValueAtQuantile(75))
	p50 := time.Duration(timings.ValueAtQuantile(50))

	if tc := timings.TotalCount(); tc > 0 {
		fmt.Println()
		fmt.Println("DNS timings,", tc, "datapoints")
		fmt.Println("\t min:\t\t", min)
		fmt.Println("\t mean:\t\t", mean)
		fmt.Println("\t [+/-sd]:\t", sd)
		fmt.Println("\t max:\t\t", max)
		fmt.Println("\t p99:\t\t", p99)
		fmt.Println("\t p95:\t\t", p95)
		fmt.Println("\t p90:\t\t", p90)
		fmt.Println("\t p75:\t\t", p75)
		fmt.Println("\t p50:\t\t", p50)

		dist := timings.Distribution()
		if *pHistDisplay && tc > 1 {
			fmt.Println()
			fmt.Println("DNS distribution,", tc, "datapoints")

			printBars(dist)
		}
	}
}

func writeBars(f *os.File, bars []hdrhistogram.Bar) {
	f.WriteString("From (ns), To (ns), Count\n")

	for _, b := range bars {
		f.WriteString(b.String())
	}
}

func printBars(bars []hdrhistogram.Bar) {
	counts := make([]int64, 0, len(bars))
	lines := make([][]string, 0, len(bars))
	added := false
	var max int64

	for _, b := range bars {
		if b.Count == 0 && !added {
			// trim the start
			continue
		}
		if b.Count > max {
			max = b.Count
		}

		added = true

		line := make([]string, 3)
		lines = append(lines, line)
		counts = append(counts, b.Count)

		line[0] = time.Duration(b.To/2 + b.From/2).String()
		line[2] = strconv.FormatInt(b.Count, 10)
	}

	for i, l := range lines {
		l[1] = makeBar(counts[i], max)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Latency", "", "Count"})
	table.SetBorder(false)
	table.AppendBulk(lines)
	table.Render()
}

func makeBar(c int64, max int64) string {
	if c == 0 {
		return ""
	}
	t := int((43 * float64(c) / float64(max)) + 0.5)
	return strings.Repeat("▄", t)
}

const fileNoBuffer = 9 // app itself needs about 9 for libs

func main() {
	version := "unknown"
	if Tag == "" {
		if Commit != "" {
			version = Commit
		}
	} else {
		version = fmt.Sprintf("%s-%s", Tag, Commit)
	}

	pApp.Version(version)
	kingpin.MustParse(pApp.Parse(os.Args[1:]))

	// process args
	color.NoColor = !*pColor

	var rLimit syscall.Rlimit

	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err == nil {
		needed := uint64(*pConcurrency) + uint64(fileNoBuffer)
		if rLimit.Cur < needed {
			fmt.Fprintf(os.Stderr, "current process limit for number of files is %d and insufficient for level of requested concurrency.\n", rLimit.Cur)
			os.Exit(2)
		}
	}

	var csv *os.File
	if *pCsv != "" {
		f, err := os.Create(*pCsv)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(2)
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
		fmt.Fprintln(os.Stderr, "Cancelling benchmark ^C, again to terminate now.")
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
