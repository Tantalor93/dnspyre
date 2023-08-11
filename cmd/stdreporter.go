package cmd

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/miekg/dns"
	"github.com/olekukonko/tablewriter"
)

type standardReporter struct{}

func (s *standardReporter) print(w io.Writer, b *Benchmark, timings *hdrhistogram.Histogram, codeTotals map[int]int64, totalCounters Counters, qtypeTotals map[string]int64, topErrs orderedMap, t time.Duration) error {
	b.printProgress(w, totalCounters)

	if len(codeTotals) > 0 {
		fmt.Println()
		fmt.Println("DNS response codes:")
		for i := dns.RcodeSuccess; i <= dns.RcodeBadCookie; i++ {
			printFn := errPrint
			if i == dns.RcodeSuccess {
				printFn = successPrint
			}
			if c, ok := codeTotals[i]; ok {
				printFn(w, "\t%s:\t%d\n", dns.RcodeToString[i], c)
			}
		}
	}

	if len(qtypeTotals) > 0 {
		fmt.Println()
		fmt.Println("DNS question types:")
		for k, v := range qtypeTotals {
			successPrint(w, "\t%s:\t%d\n", k, v)
		}
	}

	fmt.Println()

	fmt.Println("Time taken for tests:\t", highlightStr(roundDuration(t).String()))
	fmt.Printf("Questions per second:\t %s", highlightStr(fmt.Sprintf("%0.1f", float64(totalCounters.Total)/t.Seconds())))

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
		fmt.Println("DNS timings,", highlightStr(tc), "datapoints")
		fmt.Println("\t min:\t\t", highlightStr(roundDuration(min)))
		fmt.Println("\t mean:\t\t", highlightStr(roundDuration(mean)))
		fmt.Println("\t [+/-sd]:\t", highlightStr(roundDuration(sd)))
		fmt.Println("\t max:\t\t", highlightStr(roundDuration(max)))
		fmt.Println("\t p99:\t\t", highlightStr(roundDuration(p99)))
		fmt.Println("\t p95:\t\t", highlightStr(roundDuration(p95)))
		fmt.Println("\t p90:\t\t", highlightStr(roundDuration(p90)))
		fmt.Println("\t p75:\t\t", highlightStr(roundDuration(p75)))
		fmt.Println("\t p50:\t\t", highlightStr(roundDuration(p50)))

		dist := timings.Distribution()
		if b.HistDisplay && tc > 1 {
			fmt.Println()
			fmt.Println("DNS distribution,", highlightStr(tc), "datapoints")

			printBars(w, dist)
		}
	}

	sumerrs := 0
	for _, v := range topErrs.m {
		sumerrs += v
	}

	if len(topErrs.m) > 0 {
		errPrint(w, "\nTotal Errors: %d\n", sumerrs)
		errPrint(w, "Top errors:\n")
		for _, err := range topErrs.order {
			errPrint(w, "%s\t%d (%.2f)%%\n", err, topErrs.m[err], (float64(topErrs.m[err])/float64(sumerrs))*100)
		}
	}

	return nil
}

func (b *Benchmark) printProgress(w io.Writer, c Counters) {
	fmt.Printf("\nTotal requests:\t\t%s\n", highlightStr(c.Total))

	if c.IOError > 0 {
		errPrint(w, "Read/Write errors:\t%d\n", c.IOError)
	}

	if c.IDmismatch > 0 {
		errPrint(w, "ID mismatch errors:\t%d\n", c.IDmismatch)
	}

	if c.Success > 0 {
		successPrint(w, "DNS success codes:\t%d\n", c.Success)
	}

	if c.Truncated > 0 {
		errPrint(w, "Truncated responses:\t%d\n", c.Truncated)
	}
}

func printBars(w io.Writer, bars []hdrhistogram.Bar) {
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

		line[0] = roundDuration(time.Duration(b.To/2 + b.From/2)).String()
		line[2] = strconv.FormatInt(b.Count, 10)
	}

	for i, l := range lines {
		l[1] = makeBar(counts[i], max)
	}

	table := tablewriter.NewWriter(w)
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
	return strings.Repeat(highlightStr("▄"), t)
}

func roundDuration(dur time.Duration) time.Duration {
	if dur > time.Minute {
		return dur.Round(10 * time.Second)
	}
	if dur > time.Second {
		return dur.Round(10 * time.Millisecond)
	}
	if dur > time.Millisecond {
		return dur.Round(10 * time.Microsecond)
	}
	if dur > time.Microsecond {
		return dur.Round(10 * time.Nanosecond)
	}
	return dur
}
