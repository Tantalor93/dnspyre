package reporter

import (
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/miekg/dns"
	"github.com/olekukonko/tablewriter"
	"github.com/tantalor93/dnspyre/v3/pkg/dnsbench"
	"github.com/tantalor93/dnspyre/v3/pkg/printutils"
)

type standardReporter struct{}

func (s *standardReporter) print(params reportParameters) error {
	printProgress(params.outputWriter, params.totalCounters)

	if len(params.codeTotals) > 0 {
		printutils.NeutralFprintf(params.outputWriter, "\nDNS response codes:\n")
		for i := dns.RcodeSuccess; i <= dns.RcodeBadCookie; i++ {
			printFn := printutils.ErrFprintf
			if i == dns.RcodeSuccess {
				printFn = printutils.SuccessFprintf
			}
			if i == dns.RcodeNameError {
				printFn = printutils.NeutralFprintf
			}
			if c, ok := params.codeTotals[i]; ok {
				printFn(params.outputWriter, "\t%s:\t%d\n", dns.RcodeToString[i], c)
			}
		}
	}

	var dohResponseStatuses []int
	for key := range params.dohResponseStatusesTotals {
		dohResponseStatuses = append(dohResponseStatuses, key)
	}
	sort.Ints(dohResponseStatuses)

	if len(params.dohResponseStatusesTotals) > 0 {
		printutils.NeutralFprintf(params.outputWriter, "\nDoH HTTP response status codes:\n")
		for _, st := range dohResponseStatuses {
			if st == 200 {
				printutils.SuccessFprintf(params.outputWriter, "\t%d:\t%d\n", st, params.dohResponseStatusesTotals[st])
			} else {
				printutils.ErrFprintf(params.outputWriter, "\t%d:\t%d\n", st, params.dohResponseStatusesTotals[st])
			}
		}
	}

	if len(params.qtypeTotals) > 0 {
		printutils.NeutralFprintf(params.outputWriter, "\nDNS question types:\n")
		for k, v := range params.qtypeTotals {
			printutils.SuccessFprintf(params.outputWriter, "\t%s:\t%d\n", k, v)
		}
	}

	if params.benchmark.DNSSEC {
		printutils.NeutralFprintf(params.outputWriter,
			"\nNumber of domains secured using DNSSEC: %s\n", printutils.HighlightSprint(len(params.authenticatedDomains)))
	}

	printutils.NeutralFprintf(params.outputWriter, "\nTime taken for tests:\t%s\n",
		printutils.HighlightSprint(roundDuration(params.benchmarkDuration)))
	printutils.NeutralFprintf(params.outputWriter, "Questions per second:\t%s\n",
		printutils.HighlightSprintf("%0.1f", float64(params.totalCounters.Total)/params.benchmarkDuration.Seconds()))

	min := time.Duration(params.hist.Min())
	mean := time.Duration(params.hist.Mean())
	sd := time.Duration(params.hist.StdDev())
	max := time.Duration(params.hist.Max())
	p99 := time.Duration(params.hist.ValueAtQuantile(99))
	p95 := time.Duration(params.hist.ValueAtQuantile(95))
	p90 := time.Duration(params.hist.ValueAtQuantile(90))
	p75 := time.Duration(params.hist.ValueAtQuantile(75))
	p50 := time.Duration(params.hist.ValueAtQuantile(50))

	if tc := params.hist.TotalCount(); tc > 0 {
		printutils.NeutralFprintf(params.outputWriter, "DNS timings, %s datapoints\n", printutils.HighlightSprint(tc))
		printutils.NeutralFprintf(params.outputWriter, "\t min:\t\t%s\n", printutils.HighlightSprint(roundDuration(min)))
		printutils.NeutralFprintf(params.outputWriter, "\t mean:\t\t%s\n", printutils.HighlightSprint(roundDuration(mean)))
		printutils.NeutralFprintf(params.outputWriter, "\t [+/-sd]:\t%s\n", printutils.HighlightSprint(roundDuration(sd)))
		printutils.NeutralFprintf(params.outputWriter, "\t max:\t\t%s\n", printutils.HighlightSprint(roundDuration(max)))
		printutils.NeutralFprintf(params.outputWriter, "\t p99:\t\t%s\n", printutils.HighlightSprint(roundDuration(p99)))
		printutils.NeutralFprintf(params.outputWriter, "\t p95:\t\t%s\n", printutils.HighlightSprint(roundDuration(p95)))
		printutils.NeutralFprintf(params.outputWriter, "\t p90:\t\t%s\n", printutils.HighlightSprint(roundDuration(p90)))
		printutils.NeutralFprintf(params.outputWriter, "\t p75:\t\t%s\n", printutils.HighlightSprint(roundDuration(p75)))
		printutils.NeutralFprintf(params.outputWriter, "\t p50:\t\t%s\n", printutils.HighlightSprint(roundDuration(p50)))

		dist := params.hist.Distribution()
		if params.benchmark.HistDisplay && tc > 1 {
			printutils.NeutralFprintf(params.outputWriter, "\nDNS distribution, %s datapoints\n", printutils.HighlightSprint(tc))
			printBars(params.outputWriter, dist)
		}
	}

	sumerrs := 0
	for _, v := range params.topErrs.m {
		sumerrs += v
	}

	if len(params.topErrs.m) > 0 {
		printutils.ErrFprintf(params.outputWriter, "\nTotal Errors: %d\n", sumerrs)
		printutils.ErrFprintf(params.outputWriter, "Top errors:\n")
		for _, err := range params.topErrs.order {
			printutils.ErrFprintf(params.outputWriter, "%s\t%d (%.2f)%%\n", err, params.topErrs.m[err],
				(float64(params.topErrs.m[err])/float64(sumerrs))*100)
		}
	}

	return nil
}

func printProgress(w io.Writer, c dnsbench.Counters) {
	printutils.NeutralFprintf(w, "\nTotal requests:\t\t%s\n", printutils.HighlightSprint(c.Total))

	if c.IOError > 0 {
		printutils.ErrFprintf(w, "Read/Write errors:\t%d\n", c.IOError)
	}

	if c.IDmismatch > 0 {
		printutils.ErrFprintf(w, "ID mismatch errors:\t%d\n", c.IDmismatch)
	}

	if c.Success > 0 {
		printutils.SuccessFprintf(w, "DNS success responses:\t%d\n", c.Success)
	}
	if c.Negative > 0 {
		printutils.NeutralFprintf(w, "DNS negative responses:\t%d\n", c.Negative)
	}
	if c.Error > 0 {
		printutils.ErrFprintf(w, "DNS error responses:\t%d\n", c.Error)
	}

	if c.Truncated > 0 {
		printutils.ErrFprintf(w, "Truncated responses:\t%d\n", c.Truncated)
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
	return strings.Repeat(printutils.HighlightSprint("▄"), t)
}
