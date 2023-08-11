package cmd

import (
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/fatih/color"
)

var (
	errPrint     = color.New(color.FgRed).FprintfFunc()
	successPrint = color.New(color.FgGreen).FprintfFunc()
	highlightStr = color.New(color.FgYellow).SprintFunc()
)

type orderedMap struct {
	m     map[string]int
	order []string
}

// PrintReport print formatted benchmark results to stdout. If there is a fatal error while printing report, an error is returned.
func (b *Benchmark) PrintReport(w io.Writer, stats []*ResultStats, t time.Duration) error {
	// merge all the stats here
	timings := hdrhistogram.New(b.HistMin.Nanoseconds(), b.HistMax.Nanoseconds(), b.HistPre)
	codeTotals := make(map[int]int64)
	qtypeTotals := make(map[string]int64)
	times := make([]Datapoint, 0)

	errs := make(map[string]int, 0)
	top3errs := make(map[string]int)
	top3errorsInOrder := make([]string, 0)

	var totalCounters Counters

	for _, s := range stats {
		for _, err := range s.Errors {
			if v, ok := errs[err.Error()]; ok {
				errs[err.Error()] = v + 1
			} else {
				errs[err.Error()] = 1
			}
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
		now := time.Now().Format(time.RFC3339)
		dir := fmt.Sprintf("%s/graphs-%s", b.PlotDir, now)
		if err := os.Mkdir(dir, os.ModePerm); err != nil {
			panic(err)
		}
		plotHistogramLatency(b.fileName(dir, "latency-histogram"), times)
		plotBoxPlotLatency(b.fileName(dir, "latency-boxplot"), b.Server, times)
		plotResponses(b.fileName(dir, "responses-barchart"), codeTotals)
		plotLineThroughput(b.fileName(dir, "throughput-lineplot"), times)
		plotLineLatencies(b.fileName(dir, "latency-lineplot"), times)
	}

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

	if csv != nil {
		writeBars(csv, timings.Distribution())
	}

	if b.Silent {
		return nil
	}
	topErrs := orderedMap{m: top3errs, order: top3errorsInOrder}
	if b.JSON {
		j := jsonReporter{}
		return j.print(w, b, timings, codeTotals, totalCounters, qtypeTotals, topErrs, t)
	}
	s := standardReporter{}
	return s.print(w, b, timings, codeTotals, totalCounters, qtypeTotals, topErrs, t)
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
