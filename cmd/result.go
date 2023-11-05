package cmd

import (
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/miekg/dns"
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
	Duration float64
	Start    time.Time
}

// ResultStats is a representation of benchmark results of single concurrent thread.
type ResultStats struct {
	Codes                map[int]int64
	Qtypes               map[string]int64
	Hist                 *hdrhistogram.Histogram
	Timings              []Datapoint
	Counters             *Counters
	Errors               []error
	AuthenticatedDomains map[string]struct{}
}

func (rs *ResultStats) record(req *dns.Msg, resp *dns.Msg, time time.Time, timing time.Duration) {
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
	if rs.Qtypes != nil {
		rs.Qtypes[dns.TypeToString[req.Question[0].Qtype]]++
	}
	if resp.AuthenticatedData {
		if rs.AuthenticatedDomains == nil {
			rs.AuthenticatedDomains = make(map[string]struct{})
		}
		rs.AuthenticatedDomains[req.Question[0].Name] = struct{}{}
	}

	rs.Hist.RecordValue(timing.Nanoseconds())
	rs.Timings = append(rs.Timings, Datapoint{float64(timing.Milliseconds()), time})
}
