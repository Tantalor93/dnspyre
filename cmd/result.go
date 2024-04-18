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
	// Total is counter of all requests.
	Total int64
	// IOError is counter of all requests for which there was no answer.
	IOError int64
	// Success is counter of all responses which were successful (NOERROR, but not NODATA!).
	Success int64
	// Negative is counter of all responses which were negative (NODATA/NXDOMAIN).
	Negative int64
	// Error is counter of all responses which were not negative (NODATA/NXDOMAIN) or success (NOERROR response).
	Error int64
	// IDmismatch is counter of all responses which ID mismatched the request ID.
	IDmismatch int64
	// Truncated is counter of all responses which had truncated flag.
	Truncated int64
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

func newResultStats(b *Benchmark) *ResultStats {
	st := &ResultStats{Hist: hdrhistogram.New(b.HistMin.Nanoseconds(), b.HistMax.Nanoseconds(), b.HistPre)}
	if b.Rcodes {
		st.Codes = make(map[int]int64)
	}
	st.Qtypes = make(map[string]int64)
	if b.useDoH {
		st.DoHStatusCodes = make(map[int]int64)
	}
	st.Counters = &Counters{}
	return st
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
		if len(resp.Answer) == 0 {
			// NODATA negative response
			rs.Counters.Negative++
		} else {
			rs.Counters.Success++
		}
	}
	if resp.Rcode == dns.RcodeNameError {
		rs.Counters.Negative++
	}
	if resp.Rcode != dns.RcodeSuccess && resp.Rcode != dns.RcodeNameError {
		// assume every rcode not NOERROR or NXDOMAIN is error
		rs.Counters.Error++
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
