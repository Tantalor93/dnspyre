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
)

var (
	Tag    = ""
	Commit = ""
	Author = "Rahul Powar <rahul@redsift.io>"
)

var (
	pApp = kingpin.New("dnstrace", "A DTrace enabled DNS benchmark.").Author(Author)

	pDTrace  = pApp.Flag("dtrace", "Enable DTrace probes").Default("false").Bool()
	pSilent  = pApp.Flag("silent", "Disable stdout").Default("false").Bool()
	pRecurse = pApp.Flag("recurse", "Allow DNS recursion").Default("false").Bool()
	pServer  = pApp.Flag("server", "Server IP and port to query").HintOptions("8.8.8.8:53").Default("127.0.0.1").String()
	pType    = pApp.Flag("type", "Query type").Default("TXT").Enum("TXT", "A")
	pExpect  = pApp.Flag("expect", "Expect a specific response").String()

	pCount       = pApp.Flag("number", "Number of queries to issue. Note that the total number of queries issued = number*concurrency*len(queries)").Short('n').Default("1").Int64()
	pConcurrency = pApp.Flag("concurrency", "Number of concurrent queries to issue").Short('c').Default("1").Int()
	pUdpSize     = pApp.Flag("edns0", "Enable EDNS0 with specified size").Default("0").Uint16()
	pQueries     = pApp.Arg("queries", "Queries to issue.").Required().Strings()

	pWriteTimeout = pApp.Flag("write", "DNS write timeout").Default("1s").Duration()
	pReadTimeout  = pApp.Flag("read", "DNS read timeout").Default("4s").Duration()

	pHistMin     = pApp.Flag("min", "Minimum value for timing histogram in nanoseconds").Default(strconv.FormatInt(int64(time.Microsecond*100), 10)).Int64()
	pHistMax     = pApp.Flag("max", "Maximum value for histogram in nanoseconds").Default(strconv.FormatInt(int64(dnsConnectTimeout), 10)).Int64()
	pHistPre     = pApp.Flag("precision", "Significant figure for histogram precision").Default("1").Int()
	pHistDisplay = pApp.Flag("distribution", "Display distribution histogram of timings").Default("true").Bool()

	pRCodes = pApp.Flag("codes", "Enable counting DNS return codes").Default("true").Bool()

	pIOErrors = pApp.Flag("io-errors", "Log I/O errors to stderr").Default("false").Bool()

	pColor = pApp.Flag("color", "Color output").Default("true").Bool()
)

var (
	count   int64
	cerror  int64
	ecount  int64
	success int64
	matched int64
)

const dnsConnectTimeout = time.Second * 4

type rstats struct {
	codes map[int]int64
	hist  *hdrhistogram.Histogram
}

func do() []*rstats {
	questions := make([]string, len(*pQueries))
	for i, q := range *pQueries {
		questions[i] = dns.Fqdn(q)
	}

	qType := dns.TypeNone
	switch *pType {
	//TODO: Rest of them
	case "TXT":
		qType = dns.TypeTXT
	default:
		panic(fmt.Errorf("Unknown type %q", *pType))
	}

	srv := *pServer
	if strings.Index(srv, ":") == -1 {
		srv += ":53"
	}

	stats := make([]*rstats, *pConcurrency)

	var wg sync.WaitGroup

	for w := 0; w < *pConcurrency; w++ {
		st := &rstats{hist: hdrhistogram.New(*pHistMin, *pHistMax, *pHistPre)}
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

			var i int64
			for i = 1; i <= *pCount; i++ {

				for _, q := range questions {
					atomic.AddInt64(&count, 1)

					m.SetQuestion(q, qType)
					m.RecursionDesired = *pRecurse

					if co == nil {
						co, err = dns.DialTimeout("udp", srv, dnsConnectTimeout)
						if err != nil {
							atomic.AddInt64(&cerror, 1)
							continue
						}
						if udpSize := *pUdpSize; udpSize > 0 {
							m.SetEdns0(udpSize, true)
							co.UDPSize = udpSize
						}
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
						atomic.AddInt64(&success, 1)

						if expect := *pExpect; expect != "" {
							for _, s := range r.Answer {
								switch s.Header().Rrtype {
								case dns.TypeTXT:
									t := s.(*dns.TXT)
									if strings.Join(t.Txt, "") == expect {
										atomic.AddInt64(&matched, 1)
									}
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

func printReport(t time.Duration, stats []*rstats) {
	if *pSilent {
		return
	}

	// merge all the stats here
	timings := hdrhistogram.New(*pHistMin, *pHistMax, *pHistPre)
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
	successFprint(os.Stdout, "DNS success codes:\t", success, "\n")

	if *pExpect != "" {
		successFprint(os.Stdout, "Expected results:\t", matched, "\n")
	}

	if len(codeTotals) > 0 {

		fmt.Println()
		fmt.Println("DNS Codes")
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

		if *pHistDisplay {

			fmt.Println()
			fmt.Println("Distribution")

			printBars(timings.Distribution())
		}
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

	color.NoColor = !*pColor

	rand.Seed(time.Now().UnixNano())

	start := time.Now()
	res := do()
	end := time.Now()

	printReport(end.Sub(start), res)
}
