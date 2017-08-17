package main

import (
	"fmt"
	"math/rand"
	"time"

	"os"

	"strings"
	"sync"
	"sync/atomic"

	"strconv"

	"github.com/alecthomas/kingpin"
	"github.com/codahale/hdrhistogram"
	"github.com/fatih/color"
	"github.com/miekg/dns"
	"github.com/olekukonko/tablewriter"
	"syscall"
	"go.uber.org/ratelimit"
)

var (
	Tag    = ""
	Commit = ""
	Author = "Rahul Powar <rahul@redsift.io>"
)

var (
	pApp = kingpin.New("dnstrace", "A DNS benchmark.").Author(Author)

	pServer  = pApp.Flag("server", "DNS server IP:port to test.").Short('s').Default("127.0.0.1").String()
	pType    = pApp.Flag("type", "Query type.").Short('t').Default("A").Enum("TXT", "A", "AAAA") //TODO: Rest of them pt 1

	pCount       = pApp.Flag("number", "Number of queries to issue. Note that the total number of queries issued = number*concurrency*len(queries).").Short('n').Default("1").Int64()
	pConcurrency = pApp.Flag("concurrency", "Number of concurrent queries to issue.").Short('c').Default("1").Uint32()
	pRate = pApp.Flag("rate-limit", "Apply a global questions / second rate limit.").Short('l').Default("0").Int()

	pExpect  = pApp.Flag("expect", "Expect a specific response.").Short('e').Strings()


	pRecurse = pApp.Flag("recurse", "Allow DNS recursion.").Short('r').Default("false").Bool()
	pUdpSize     = pApp.Flag("edns0", "Enable EDNS0 with specified size.").Default("0").Uint16()
	pTCP = pApp.Flag("tcp", "Use TCP fot DNS requests.").Default("false").Bool()

	pWriteTimeout = pApp.Flag("write", "DNS write timeout.").Default("1s").Duration()
	pReadTimeout  = pApp.Flag("read", "DNS read timeout.").Default(dnsTimeout.String()).Duration()

	pRCodes = pApp.Flag("codes", "Enable counting DNS return codes.").Default("true").Bool()

	pHistMin     = pApp.Flag("min", "Minimum value for timing histogram.").Default((time.Microsecond*400).String()).Duration()
	pHistMax     = pApp.Flag("max", "Maximum value for histogram.").Default(dnsTimeout.String()).Duration()
	pHistPre     = pApp.Flag("precision", "Significant figure for histogram precision.").Default("1").PlaceHolder("[1-5]").Int()
	pHistDisplay = pApp.Flag("distribution", "Display distribution histogram of timings to stdout.").Default("true").Bool()
	pCsv = pApp.Flag("csv", "Export distribution to CSV.").Default("").PlaceHolder("/path/to/file.csv").String()

	pIOErrors = pApp.Flag("io-errors", "Log I/O errors to stderr.").Default("false").Bool()

	pSilent  = pApp.Flag("silent", "Disable stdout.").Default("false").Bool()
	pColor = pApp.Flag("color", "ANSI Color output.").Default("true").Bool()

	pQueries     = pApp.Arg("queries", "Queries to issue.").Required().Strings()
)

var (
	count   int64
	cerror  int64
	ecount  int64
	success int64
	matched int64
	mismatch int64
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

func do() []*rstats {
	questions := make([]string, len(*pQueries))
	for i, q := range *pQueries {
		questions[i] = dns.Fqdn(q)
	}

	qType := dns.TypeNone
	switch *pType {
	//TODO: Rest of them pt 2
	case "TXT":
		qType = dns.TypeTXT
	case "A":
		qType = dns.TypeA
	case "AAAA":
		qType = dns.TypeAAAA
	default:
		panic(fmt.Errorf("Unknown type %q", *pType))
	}

	srv := *pServer
	if strings.Index(srv, ":") == -1 {
		srv += ":53"
	}

	network := "udp"
	if *pTCP {
		network = "tcp"
	}

	conncurrent := *pConcurrency

	limits := ""
	var limit ratelimit.Limiter
	if *pRate > 0 {
		limit = ratelimit.New(*pRate)
		limits = fmt.Sprintf("(limited to %d QPS)", *pRate)
	}

	if !*pSilent {
		fmt.Printf("Benchmarking %s via %s with %d conncurrent requests %s\n\n", srv, network, conncurrent, limits)

	}



	stats := make([]*rstats, conncurrent)

	var wg sync.WaitGroup
	var w uint32
	for w = 0; w < conncurrent; w++ {
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
			question := dns.Question{"", qType, dns.ClassINET}

			// create a new lock free rand source for this goroutine
			rando := rand.New(rand.NewSource(time.Now().Unix()))

			var i int64
			for i = 1; i <= *pCount; i++ {

				for _, q := range questions {
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
								fmt.Fprintln(os.Stderr,"i/o error dialing: ", err.Error())
							}
							continue
						}
						if udpSize := *pUdpSize; udpSize > 0 {
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
							fmt.Fprintln(os.Stderr,"i/o error writing: ", err.Error())
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
							fmt.Fprintln(os.Stderr,"i/o error reading: ", err.Error())
						}
						co.Close()
						co = nil
						continue
					}
					timing := time.Now().Sub(start)

					st.hist.RecordValue(timing.Nanoseconds())

					if r.Rcode == dns.RcodeSuccess {
						if r.Id != m.Id {
							atomic.AddInt64(&mismatch, 1)
							continue
						}
						atomic.AddInt64(&success, 1)

						if expect := *pExpect; len(expect) > 0 {
							for _, s := range r.Answer {
								ok := false
								switch s.Header().Rrtype {
								//TODO: Rest of them pt 3
								case dns.TypeA:
									a := s.(*dns.A)
									ok = isExpected(a.A.To4().String())

								case dns.TypeAAAA:
									a := s.(*dns.A)
									ok = isExpected(a.A.String())

								case dns.TypeTXT:
									t := s.(*dns.TXT)
									ok = isExpected(strings.Join(t.Txt, ""))
								}

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

func printReport(t time.Duration, stats []*rstats, csv *os.File) {
	defer func() {
		if csv != nil {
			csv.Close()
		}
	}()

	if *pSilent {
		return
	}

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

	errorFprint := color.New(color.FgRed).Fprint
	successFprint := color.New(color.FgGreen).Fprint
	infoFprint := color.New().Fprint

	infoFprint(os.Stdout, "Total requests:\t\t", count, "\n")
	if cerror > 0 || ecount > 0 {
		errorFprint(os.Stdout, "Connection errors:\t", cerror, "\n")
		errorFprint(os.Stdout, "Read/Write errors:\t", ecount, "\n")
	}

	if mismatch > 0 {
		errorFprint(os.Stdout, "ID mismatch errors:\t", mismatch, "\n")
	}

	successFprint(os.Stdout, "DNS success codes:\t", success, "\n")

	if len(*pExpect) > 0 {
		successFprint(os.Stdout, "Expected results:\t", matched, "\n")
	}

	if len(codeTotals) > 0 {

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

	if tc := timings.TotalCount(); tc > 0 {
		fmt.Println()
		fmt.Println("DNS timings,", tc, "datapoints")
		fmt.Println("\t min:\t\t", min)
		fmt.Println("\t mean:\t\t", mean)
		fmt.Println("\t [+/-sd]:\t", sd)
		fmt.Println("\t max:\t\t", max)

		dist := timings.Distribution()
		if *pHistDisplay && tc > 1 {

			fmt.Println()
			fmt.Println("DNS distribution,", tc, "datapoints")

			printBars(dist)
		}

		if csv != nil {

			writeBars(csv, dist)

			fmt.Println()
			fmt.Println("DNS distribution written to", csv.Name())
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
		var needed uint64
		needed = uint64(*pConcurrency) + uint64(fileNoBuffer)
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

	// get going

	rand.Seed(time.Now().UnixNano())

	start := time.Now()
	res := do()
	end := time.Now()

	printReport(end.Sub(start), res, csv)
}
