package dnstrace

import (
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/codahale/hdrhistogram"
	"github.com/fatih/color"
	"github.com/miekg/dns"
	"github.com/olekukonko/tablewriter"
)

func printProgress() {
	if *pSilent {
		return
	}

	logger.Println()

	errorFprint := color.New(color.FgRed).Fprint
	successFprint := color.New(color.FgGreen).Fprint

	acount := atomic.LoadInt64(&count)
	acerror := atomic.LoadInt64(&cerror)
	aecount := atomic.LoadInt64(&ecount)
	amismatch := atomic.LoadInt64(&mismatch)
	asuccess := atomic.LoadInt64(&success)
	amatched := atomic.LoadInt64(&matched)
	atruncated := atomic.LoadInt64(&truncated)

	logger.Printf("Total requests:\t %d\t", acount)

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

		logger.Println()
		logger.Println("DNS distribution written to", csv.Name())
	}

	if *pSilent {
		return
	}

	printProgress()

	if len(codeTotals) > 0 {
		errorFprint := color.New(color.FgRed).Fprint
		successFprint := color.New(color.FgGreen).Fprint

		logger.Println()
		logger.Println("DNS response codes")
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

	logger.Println()

	logger.Println("Time taken for tests:\t", t.String())
	logger.Printf("Questions per second:\t %0.1f", float64(count)/t.Seconds())

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
		logger.Println()
		logger.Println("DNS timings,", tc, "datapoints")
		logger.Println("\t min:\t\t", min)
		logger.Println("\t mean:\t\t", mean)
		logger.Println("\t [+/-sd]:\t", sd)
		logger.Println("\t max:\t\t", max)
		logger.Println("\t p99:\t\t", p99)
		logger.Println("\t p95:\t\t", p95)
		logger.Println("\t p90:\t\t", p90)
		logger.Println("\t p75:\t\t", p75)
		logger.Println("\t p50:\t\t", p50)

		dist := timings.Distribution()
		if *pHistDisplay && tc > 1 {
			logger.Println()
			logger.Println("DNS distribution,", tc, "datapoints")

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
