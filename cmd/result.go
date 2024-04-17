package cmd

import (
	"errors"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/miekg/dns"
	"github.com/tantalor93/doh-go/doh"
)

// Counters represents various counters of benchmark results.
type Counters struct {
	Total      int64
	IOError    int64
	Success    int64
	IDmismatch int64
	Truncated  int64
}

// Datapoint one datapoint of benchmark (single DNS request).
type Datapoint struct {
	Duration time.Duration
	Start    time.Time
}

// ErrorDatapoint one datapoint representing single IO error of benchmark.
// Datapoint one datapoint of benchmark (single DNS request).
type ErrorDatapoint struct {
	Start time.Time
	Err   error
}

// ResultStats is a representation of benchmark results of single concurrent thread.
type ResultStats struct {
	Codes                map[int]int64
	Qtypes               map[string]int64
	Hist                 *hdrhistogram.Histogram
	Timings              []Datapoint
	Counters             *Counters
	Errors               []ErrorDatapoint
	AuthenticatedDomains map[string]struct{}
	DoHStatusCodes       map[int]int64
}

func (rs *ResultStats) record(req *dns.Msg, resp *dns.Msg, err error, time time.Time, duration time.Duration) {
	rs.Counters.Total++

	if rs.DoHStatusCodes != nil {
		statusError := doh.UnexpectedServerHTTPStatusError{}
		if err != nil && errors.As(err, &statusError) {
			rs.DoHStatusCodes[statusError.HTTPStatus()]++
		}
		if err == nil {
			rs.DoHStatusCodes[200]++
		}
	}

	if rs.Qtypes != nil {
		rs.Qtypes[dns.TypeToString[req.Question[0].Qtype]]++
	}

	if err != nil {
		rs.Counters.IOError++
		rs.Errors = append(rs.Errors, ErrorDatapoint{Start: time, Err: err})
		return
	}

	if resp.Truncated {
		rs.Counters.Truncated++
	}

	if resp.Rcode == dns.RcodeSuccess {
		if resp.Id != req.Id {
			rs.Counters.IDmismatch++
			return
		}
		rs.Counters.Success++
	}

	if rs.Codes != nil {
		var c int64
		if v, ok := rs.Codes[resp.Rcode]; ok {
			c = v
		}
		c++
		rs.Codes[resp.Rcode] = c
	}
	if resp.AuthenticatedData {
		if rs.AuthenticatedDomains == nil {
			rs.AuthenticatedDomains = make(map[string]struct{})
		}
		rs.AuthenticatedDomains[req.Question[0].Name] = struct{}{}
	}

	rs.Hist.RecordValue(duration.Nanoseconds())
	rs.Timings = append(rs.Timings, Datapoint{Duration: duration, Start: time})
}
