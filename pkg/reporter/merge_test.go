package reporter_test

import (
	"errors"
	"testing"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/tantalor93/dnspyre/v3/pkg/dnsbench"
	"github.com/tantalor93/dnspyre/v3/pkg/reporter"
)

func TestMerge(t *testing.T) {
	start := time.Now()
	stats := []*dnsbench.ResultStats{
		{
			Codes: map[int]int64{
				dns.RcodeSuccess:       2,
				dns.RcodeNameError:     1,
				dns.RcodeServerFailure: 1,
			},
			Qtypes: map[string]int64{
				"A":    2,
				"AAAA": 2,
			},
			Hist: histogramWithValues(time.Second, 2*time.Second, time.Second),
			Timings: []dnsbench.Datapoint{
				{
					Start:    start,
					Duration: time.Second,
				},
				{
					Start:    start.Add(time.Second),
					Duration: 2 * time.Second,
				},
				{
					Start:    start.Add(2 * time.Second),
					Duration: time.Second,
				},
			},
			Counters: &dnsbench.Counters{
				Success:    2,
				Negative:   1,
				Truncated:  1,
				IOError:    2,
				Error:      1,
				IDmismatch: 1,
				Total:      8,
			},
			Errors: []dnsbench.ErrorDatapoint{
				{
					Start: start.Add(3 * time.Second),
					Err:   errors.New("test"),
				},
				{
					Start: start.Add(4 * time.Second),
					Err:   errors.New("test"),
				},
			},
			AuthenticatedDomains: map[string]struct{}{
				"google.com.": {},
			},
			DoHStatusCodes: map[int]int64{
				200: 5,
				503: 1,
			},
		},
		{
			Codes: map[int]int64{
				dns.RcodeSuccess:       1,
				dns.RcodeNameError:     1,
				dns.RcodeServerFailure: 1,
			},
			Qtypes: map[string]int64{
				"A":    1,
				"AAAA": 2,
			},
			Hist: histogramWithValues(time.Second, 2*time.Second),
			Timings: []dnsbench.Datapoint{
				{
					Start:    start.Add(time.Second),
					Duration: time.Second,
				},
				{
					Start:    start.Add(3 * time.Second),
					Duration: 2 * time.Second,
				},
			},
			Counters: &dnsbench.Counters{
				Success:    1,
				Negative:   1,
				Truncated:  1,
				IOError:    1,
				Error:      1,
				IDmismatch: 1,
				Total:      6,
			},
			Errors: []dnsbench.ErrorDatapoint{
				{
					Start: start.Add(3 * time.Second),
					Err:   errors.New("test2"),
				},
			},
			AuthenticatedDomains: map[string]struct{}{
				"google.com.": {},
			},
			DoHStatusCodes: map[int]int64{
				200: 4,
				500: 1,
			},
		},
	}

	want := reporter.BenchmarkResultStats{
		Codes: map[int]int64{
			dns.RcodeSuccess:       3,
			dns.RcodeNameError:     2,
			dns.RcodeServerFailure: 2,
		},
		Qtypes: map[string]int64{
			"A":    3,
			"AAAA": 4,
		},
		Hist: histogramWithValues(time.Second, 2*time.Second, time.Second, time.Second, 2*time.Second),
		Timings: []dnsbench.Datapoint{
			{
				Start:    start,
				Duration: time.Second,
			},
			{
				Start:    start.Add(time.Second),
				Duration: 2 * time.Second,
			},
			{
				Start:    start.Add(time.Second),
				Duration: time.Second,
			},
			{
				Start:    start.Add(2 * time.Second),
				Duration: time.Second,
			},
			{
				Start:    start.Add(3 * time.Second),
				Duration: 2 * time.Second,
			},
		},
		Counters: dnsbench.Counters{
			Success:    3,
			Negative:   2,
			Truncated:  2,
			IOError:    3,
			Error:      2,
			IDmismatch: 2,
			Total:      14,
		},
		Errors: []dnsbench.ErrorDatapoint{
			{
				Start: start.Add(3 * time.Second),
				Err:   errors.New("test"),
			},
			{
				Start: start.Add(3 * time.Second),
				Err:   errors.New("test2"),
			},
			{
				Start: start.Add(4 * time.Second),
				Err:   errors.New("test"),
			},
		},
		GroupedErrors: map[string]int{
			"test":  2,
			"test2": 1,
		},
		AuthenticatedDomains: map[string]struct{}{
			"google.com.": {},
		},
		DoHStatusCodes: map[int]int64{
			200: 9,
			500: 1,
			503: 1,
		},
	}

	res := reporter.Merge(&dnsbench.Benchmark{DNSSEC: true, HistMin: 0, HistMax: 5 * time.Second, HistPre: 1}, stats)

	assert.Equal(t, want, res)
}

func histogramWithValues(durations ...time.Duration) *hdrhistogram.Histogram {
	hst := hdrhistogram.New(0, 5*time.Second.Nanoseconds(), 1)
	for _, v := range durations {
		hst.RecordValue(v.Nanoseconds())
	}
	return hst
}
