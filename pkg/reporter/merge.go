package reporter

import (
	"errors"
	"net"
	"sort"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/tantalor93/dnspyre/v3/pkg/dnsbench"
)

// BenchmarkResultStats represents merged results of the dnsbench.Benchmark execution.
type BenchmarkResultStats struct {
	Codes                map[int]int64
	Qtypes               map[string]int64
	Hist                 *hdrhistogram.Histogram
	Timings              []dnsbench.Datapoint
	Counters             dnsbench.Counters
	Errors               []dnsbench.ErrorDatapoint
	GroupedErrors        map[string]int
	AuthenticatedDomains map[string]struct{}
	DoHStatusCodes       map[int]int64
}

// Merge takes results of the executed dnsbench.Benchmark and merges them.
func Merge(b *dnsbench.Benchmark, stats []*dnsbench.ResultStats) BenchmarkResultStats {
	totals := BenchmarkResultStats{
		Codes:                make(map[int]int64),
		Qtypes:               make(map[string]int64),
		Hist:                 hdrhistogram.New(b.HistMin.Nanoseconds(), b.HistMax.Nanoseconds(), b.HistPre),
		GroupedErrors:        make(map[string]int),
		AuthenticatedDomains: make(map[string]struct{}),
		DoHStatusCodes:       make(map[int]int64),
	}

	for _, s := range stats {
		for _, err := range s.Errors {
			errorString := errString(err)

			if v, ok := totals.GroupedErrors[errorString]; ok {
				totals.GroupedErrors[errorString] = v + 1
			} else {
				totals.GroupedErrors[errorString] = 1
			}
		}
		totals.Errors = append(totals.Errors, s.Errors...)

		totals.Hist.Merge(s.Hist)
		totals.Timings = append(totals.Timings, s.Timings...)
		if s.Codes != nil {
			for k, v := range s.Codes {
				totals.Codes[k] += v
			}
		}
		if s.Qtypes != nil {
			for k, v := range s.Qtypes {
				totals.Qtypes[k] += v
			}
		}
		if s.DoHStatusCodes != nil {
			for k, v := range s.DoHStatusCodes {
				totals.DoHStatusCodes[k] += v
			}
		}
		if s.Counters != nil {
			totals.Counters = dnsbench.Counters{
				Total:      totals.Counters.Total + s.Counters.Total,
				IOError:    totals.Counters.IOError + s.Counters.IOError,
				Success:    totals.Counters.Success + s.Counters.Success,
				Negative:   totals.Counters.Negative + s.Counters.Negative,
				Error:      totals.Counters.Error + s.Counters.Error,
				IDmismatch: totals.Counters.IDmismatch + s.Counters.IDmismatch,
				Truncated:  totals.Counters.Truncated + s.Counters.Truncated,
			}
		}
		if b.DNSSEC {
			for k := range s.AuthenticatedDomains {
				totals.AuthenticatedDomains[k] = struct{}{}
			}
		}
	}

	// sort data points from the oldest to the earliest, so we can better plot time dependant graphs (like line)
	sort.SliceStable(totals.Timings, func(i, j int) bool {
		return totals.Timings[i].Start.Before(totals.Timings[j].Start)
	})

	// sort error data points from the oldest to the earliest, so we can better plot time dependant graphs (like line)
	sort.SliceStable(totals.Errors, func(i, j int) bool {
		return totals.Errors[i].Start.Before(totals.Errors[j].Start)
	})
	return totals
}

func errString(err dnsbench.ErrorDatapoint) string {
	var errorString string
	var netOpErr *net.OpError
	var resolveErr *net.DNSError

	switch {
	case errors.As(err.Err, &resolveErr):
		errorString = resolveErr.Err + " " + resolveErr.Name
	case errors.As(err.Err, &netOpErr):
		errorString = netOpErr.Op + " " + netOpErr.Net
		if netOpErr.Addr != nil {
			errorString += " " + netOpErr.Addr.String()
		}
	default:
		errorString = err.Err.Error()
	}
	return errorString
}
