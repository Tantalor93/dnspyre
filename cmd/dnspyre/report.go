package dnspyre

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/fatih/color"
	"github.com/miekg/dns"
	"github.com/olekukonko/tablewriter"
)

var (
	errPrint     = color.New(color.FgRed).FprintfFunc()
	successPrint = color.New(color.FgGreen).FprintfFunc()
	highlightStr = color.New(color.FgYellow).SprintFunc()
)

func (b *Benchmark) printProgress(c Counters) {
	fmt.Printf("\nTotal requests:\t\t%s\n", highlightStr(c.Total))

	if c.ConnError > 0 {
		errPrint(os.Stdout, "Connection errors:\t%d\n", c.ConnError)
	}
	if c.IOError > 0 {
		errPrint(os.Stdout, "Read/Write errors:\t%d\n", c.IOError)
	}

	if c.IDmismatch > 0 {
		errPrint(os.Stdout, "ID mismatch errors:\t%d\n", c.IDmismatch)
	}

	if c.Success > 0 {
		successPrint(os.Stdout, "DNS success codes:\t%d\n", c.Success)
	}

	if c.Truncated > 0 {
		errPrint(os.Stdout, "Truncated responses:\t%d\n", c.Truncated)
	}
}

// PrintReport print formatted benchmark results to stdout. If there is a fatal error while printing report, an error is returned.
func (b *Benchmark) PrintReport(stats []*ResultStats, t time.Duration) error {
	var csv *os.File
	if b.Csv != "" {
		f, err := os.Create(b.Csv)
		if err != nil {
			return fmt.Errorf("failed to create file for CSV export due to '%v'", err)
		}

		csv = f
	}

	defer func() {
		if csv != nil {
			csv.Close()
		}
	}()

	// merge all the stats here
	timings := hdrhistogram.New(b.HistMin.Nanoseconds(), b.HistMax.Nanoseconds(), b.HistPre)
	codeTotals := make(map[int]int64)
	qtypeTotals := make(map[string]int64)
	times := make([]Datapoint, 0)

	errs := make(map[string]int, 0)
	top3errs := make(map[string]int)
	top3errorsInOrder := make([]string, 0)
	sumerrs := 0

	var totalCounters Counters

	for _, s := range stats {
		for _, err := range s.Errors {
			if v, ok := errs[err.Error()]; ok {
				errs[err.Error()] = v + 1
			} else {
				errs[err.Error()] = 1
			}
			sumerrs++
		}

		timings.Merge(s.Hist)
		times = append(times, s.Timings...)
		if s.Codes != nil {
			for k, v := range s.Codes {
				codeTotals[k] += v
			}
		}
		if s.Qtypes != nil {
			for k, v := range s.Qtypes {
				qtypeTotals[k] += v
			}
		}
		if s.Counters != nil {
			totalCounters = Counters{
				Total:      totalCounters.Total + s.Counters.Total,
				ConnError:  totalCounters.ConnError + s.Counters.ConnError,
				IOError:    totalCounters.IOError + s.Counters.IOError,
				Success:    totalCounters.Success + s.Counters.Success,
				IDmismatch: totalCounters.IDmismatch + s.Counters.IDmismatch,
				Truncated:  totalCounters.Truncated + s.Counters.Truncated,
			}
		}
	}

	for i := 0; i < 3; i++ {
		max := 0
		maxerr := ""
		for k, v := range errs {
			if _, ok := top3errs[k]; v > max && !ok {
				maxerr = k
				max = v
			}
		}
		if max != 0 {
			top3errs[maxerr] = max
			top3errorsInOrder = append(top3errorsInOrder, maxerr)
		}
	}

	// sort data points from the oldest to the earliest so we can better plot time dependant graphs (like line)
	sort.SliceStable(times, func(i, j int) bool {
		return times[i].Start.Before(times[j].Start)
	})

	if len(b.PlotDir) != 0 {
		now := time.Now()
		unix := now.Unix()
		dir := fmt.Sprintf("%s/graphs-%d", b.PlotDir, unix)
		if err := os.Mkdir(dir, os.ModePerm); err != nil {
			panic(err)
		}
		plotHistogramLatency(b.fileName(dir, "latency-histogram"), times)
		plotBoxPlotLatency(b.fileName(dir, "latency-boxplot"), b.Server, times)
		plotResponses(b.fileName(dir, "responses-barchart"), codeTotals)
		plotLineThroughput(b.fileName(dir, "throughput-lineplot"), times)
		plotLineLatencies(b.fileName(dir, "latency-lineplot"), times)
	}

	if csv != nil {
		writeBars(csv, timings.Distribution())

		if !b.Silent {
			fmt.Println()
			fmt.Println("DNS distribution written to", csv.Name())
		}
	}

	if b.Silent {
		return nil
	}

	b.printProgress(totalCounters)

	if len(codeTotals) > 0 {
		fmt.Println()
		fmt.Println("DNS response codes:")
		for i := dns.RcodeSuccess; i <= dns.RcodeBadCookie; i++ {
			printFn := errPrint
			if i == dns.RcodeSuccess {
				printFn = successPrint
			}
			if c, ok := codeTotals[i]; ok {
				printFn(os.Stdout, "\t%s:\t%d\n", dns.RcodeToString[i], c)
			}
		}
	}

	if len(qtypeTotals) > 0 {
		fmt.Println()
		fmt.Println("DNS question types:")
		for k, v := range qtypeTotals {
			successPrint(os.Stdout, "\t%s:\t%d\n", k, v)
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
			fmt.Println("DNS distribution,", tc, "datapoints")

			printBars(dist)
		}
	}

	if len(top3errs) > 0 {
		errPrint(os.Stdout, "\nTotal Errors: %d\n", sumerrs)
		errPrint(os.Stdout, "Top errors:\n")
		for _, err := range top3errorsInOrder {
			errPrint(os.Stdout, "%s\t%d (%.2f)%%\n", err, top3errs[err], (float64(top3errs[err])/float64(sumerrs))*100)
		}
	}
	return nil
}

func (b *Benchmark) fileName(dir, name string) string {
	return dir + "/" + name + "." + b.PlotFormat
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

		line[0] = roundDuration(time.Duration(b.To/2 + b.From/2)).String()
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

func roundDuration(dur time.Duration) time.Duration {
	if dur > time.Minute {
		return dur.Round(100 * time.Second)
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
