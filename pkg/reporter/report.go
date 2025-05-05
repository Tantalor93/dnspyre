package reporter

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/tantalor93/dnspyre/v3/pkg/dnsbench"
)

type orderedMap struct {
	m     map[string]int
	order []string
}

type reportParameters struct {
	benchmark                 *dnsbench.Benchmark
	outputWriter              io.Writer
	hist                      *hdrhistogram.Histogram
	codeTotals                map[int]int64
	totalCounters             dnsbench.Counters
	qtypeTotals               map[string]int64
	topErrs                   orderedMap
	authenticatedDomains      map[string]struct{}
	benchmarkDuration         time.Duration
	dohResponseStatusesTotals map[int]int64
}

type reportPrinter interface {
	print(params reportParameters) error
}

// PrintReport prints formatted benchmark result to stdout, exports graphs and generates CSV output if configured.
// If there is a fatal error while printing report, an error is returned.
func PrintReport(b *dnsbench.Benchmark, stats []*dnsbench.ResultStats, benchStart time.Time, benchDuration time.Duration) error {
	totals := Merge(b, stats)

	top3errs := make(map[string]int)
	top3errorsInOrder := make([]string, 0)

	for i := 0; i < 3; i++ {
		maxerr := 0
		maxerrstr := ""
		for k, v := range totals.GroupedErrors {
			if _, ok := top3errs[k]; v > maxerr && !ok {
				maxerrstr = k
				maxerr = v
			}
		}
		if maxerr != 0 {
			top3errs[maxerrstr] = maxerr
			top3errorsInOrder = append(top3errorsInOrder, maxerrstr)
		}
	}

	if len(b.PlotDir) != 0 {
		if err := directoryExists(b.PlotDir); err != nil {
			return fmt.Errorf("unable to plot results: %w", err)
		}

		now := time.Now().Format(time.RFC3339)
		dir := fmt.Sprintf("%s/graphs-%s", b.PlotDir, now)
		if err := os.Mkdir(dir, os.ModePerm); err != nil {
			return fmt.Errorf("unable to plot results: %w", err)
		}
		plotHistogramLatency(fileName(b, dir, "latency-histogram"), totals.Timings)
		plotBoxPlotLatency(fileName(b, dir, "latency-boxplot"), b.Server, totals.Timings)
		plotResponses(fileName(b, dir, "responses-barchart"), totals.Codes)
		plotLineThroughput(fileName(b, dir, "throughput-lineplot"), benchStart, totals.Timings)
		plotLineLatencies(fileName(b, dir, "latency-lineplot"), benchStart, totals.Timings)
		plotErrorRate(fileName(b, dir, "errorrate-lineplot"), benchStart, totals.Errors)
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
		writeBars(csv, totals.Hist.Distribution())
	}

	if b.Silent {
		return nil
	}
	topErrs := orderedMap{m: top3errs, order: top3errorsInOrder}
	params := reportParameters{
		benchmark:                 b,
		outputWriter:              b.Writer,
		hist:                      totals.Hist,
		codeTotals:                totals.Codes,
		totalCounters:             totals.Counters,
		qtypeTotals:               totals.Qtypes,
		topErrs:                   topErrs,
		authenticatedDomains:      totals.AuthenticatedDomains,
		benchmarkDuration:         benchDuration,
		dohResponseStatusesTotals: totals.DoHStatusCodes,
	}
	return printer(b).print(params)
}

func directoryExists(plotDir string) error {
	stat, err := os.Stat(plotDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("'%s' path does not point to an existing directory", plotDir)
		}
		return err
	} else if !stat.IsDir() {
		return fmt.Errorf("'%s' is not a path to a directory", plotDir)
	}
	return nil
}

func printer(b *dnsbench.Benchmark) reportPrinter {
	switch {
	case b.JSON:
		return &jsonReporter{}
	default:
		return &standardReporter{}
	}
}

func fileName(b *dnsbench.Benchmark, dir, name string) string {
	return dir + "/" + name + "." + b.PlotFormat
}

func writeBars(f *os.File, bars []hdrhistogram.Bar) {
	f.WriteString("From (ns), To (ns), Count\n")

	for _, b := range bars {
		f.WriteString(b.String())
	}
}
