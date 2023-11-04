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

func (s *standardReporter) print(params reportParameters) error {
	params.benchmark.printProgress(params.outputWriter, params.totalCounters)

	if len(params.codeTotals) > 0 {
		fmt.Println()
		fmt.Println("DNS response codes:")
		for i := dns.RcodeSuccess; i <= dns.RcodeBadCookie; i++ {
			printFn := errPrint
			if i == dns.RcodeSuccess {
				printFn = successPrint
			}
			if c, ok := params.codeTotals[i]; ok {
				printFn(params.outputWriter, "\t%s:\t%d\n", dns.RcodeToString[i], c)
			}
		}
	}

	if len(params.qtypeTotals) > 0 {
		fmt.Println()
		fmt.Println("DNS question types:")
		for k, v := range params.qtypeTotals {
			successPrint(params.outputWriter, "\t%s:\t%d\n", k, v)
		}
	}

	if params.benchmark.DNSSEC {
		fmt.Println()
		fmt.Println("Number of domains secured using DNSSEC:", highlightStr(len(params.authenticatedDomains)))
	}

	fmt.Println()

	fmt.Println("Time taken for tests:\t", highlightStr(roundDuration(params.benchmarkDuration).String()))
	fmt.Printf("Questions per second:\t %s", highlightStr(fmt.Sprintf("%0.1f", float64(params.totalCounters.Total)/params.benchmarkDuration.Seconds())))

	min := time.Duration(params.timings.Min())
	mean := time.Duration(params.timings.Mean())
	sd := time.Duration(params.timings.StdDev())
	max := time.Duration(params.timings.Max())
	p99 := time.Duration(params.timings.ValueAtQuantile(99))
	p95 := time.Duration(params.timings.ValueAtQuantile(95))
	p90 := time.Duration(params.timings.ValueAtQuantile(90))
	p75 := time.Duration(params.timings.ValueAtQuantile(75))
	p50 := time.Duration(params.timings.ValueAtQuantile(50))

	if tc := params.timings.TotalCount(); tc > 0 {
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

		dist := params.timings.Distribution()
		if params.benchmark.HistDisplay && tc > 1 {
			fmt.Println()
			fmt.Println("DNS distribution,", highlightStr(tc), "datapoints")

			printBars(params.outputWriter, dist)
		}
	}

	sumerrs := 0
	for _, v := range params.topErrs.m {
		sumerrs += v
	}

	if len(params.topErrs.m) > 0 {
		errPrint(params.outputWriter, "\nTotal Errors: %d\n", sumerrs)
		errPrint(params.outputWriter, "Top errors:\n")
		for _, err := range params.topErrs.order {
			errPrint(params.outputWriter, "%s\t%d (%.2f)%%\n", err, params.topErrs.m[err], (float64(params.topErrs.m[err])/float64(sumerrs))*100)
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
