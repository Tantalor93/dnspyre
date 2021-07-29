package dnstrace

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/fatih/color"
	"github.com/miekg/dns"
	"github.com/olekukonko/tablewriter"
)

var (
	errPrint     = color.New(color.FgRed).Fprint
	successPrint = color.New(color.FgGreen).Fprint
)

func printProgress() {
	if *pSilent {
		return
	}

	fmt.Println()

	acount := atomic.LoadInt64(&count)
	acerror := atomic.LoadInt64(&cerror)
	aecount := atomic.LoadInt64(&ecount)
	amismatch := atomic.LoadInt64(&mismatch)
	asuccess := atomic.LoadInt64(&success)
	amatched := atomic.LoadInt64(&matched)
	atruncated := atomic.LoadInt64(&truncated)

	fmt.Printf("Total requests:\t\t%d\t\n", acount)

	if acerror > 0 || aecount > 0 {
		errPrint(os.Stdout, "Connection errors:\t", acerror, "\n")
		errPrint(os.Stdout, "Read/Write errors:\t", aecount, "\n")
	}

	if amismatch > 0 {
		errPrint(os.Stdout, "ID mismatch errors:\t", amismatch, "\n")
	}

	successPrint(os.Stdout, "DNS success codes:\t", asuccess, "\n")

	if atruncated > 0 {
		errPrint(os.Stdout, "Truncated responses:\t", atruncated, "\n")
	} else {
		successPrint(os.Stdout, "Truncated responses:\t", atruncated, "\n")
	}

	if len(*pExpect) > 0 {
		expect := successPrint
		if amatched != asuccess {
			expect = errPrint
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
	times := make([]datapoint, 0)
	for _, s := range stats {
		timings.Merge(s.hist)
		times = append(times, s.timings...)
		if s.codes != nil {
			for k, v := range s.codes {
				codeTotals[k] = codeTotals[k] + v
			}
		}
	}

	// sort data points from the oldest to the earliest so we can better plot time dependant graphs (like line)
	sort.SliceStable(times, func(i, j int) bool {
		return times[i].start.Before(times[j].start)
	})

	if len(*pPlotDir) != 0 {
		now := time.Now()
		unix := now.Unix()
		plotHistogramLatency(getFileName("latency-hist", unix), times)
		plotBoxPlotLatency(getFileName("latency-box", unix), *pServer, times)
		plotLineLatency(getFileName("latency-line", unix), times)
		plotResponses(getFileName("responses-bar", unix), codeTotals)
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
		fmt.Println()
		fmt.Println("DNS response codes:")
		for i := dns.RcodeSuccess; i <= dns.RcodeBadCookie; i++ {
			printFn := errPrint
			if i == dns.RcodeSuccess {
				printFn = successPrint
			}
			if c, ok := codeTotals[i]; ok {
				printFn(os.Stdout, "\t", dns.RcodeToString[i]+":\t", c, "\n")
			}
		}
	}

	fmt.Println()

	fmt.Println("Time taken for tests:\t", t.String())
	fmt.Printf("Questions per second:\t %0.1f", float64(count)/t.Seconds())

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

func getFileName(filePrefix string, unix int64) string {
	return *pPlotDir + "/" + filePrefix + "-" + strconv.FormatInt(unix, 10) + "." + *pPlotFormat
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
