package cmd

import (
	"encoding/json"
	"math"
	"time"

	"github.com/miekg/dns"
)

type jsonReporter struct{}

type latencyStats struct {
	MinMs  int64 `json:"minMs"`
	MeanMs int64 `json:"meanMs"`
	StdMs  int64 `json:"stdMs"`
	MaxMs  int64 `json:"maxMs"`
	P99Ms  int64 `json:"p99Ms"`
	P95Ms  int64 `json:"p95Ms"`
	P90Ms  int64 `json:"p90Ms"`
	P75Ms  int64 `json:"p75Ms"`
	P50Ms  int64 `json:"p50Ms"`
}

type histogramPoint struct {
	LatencyMs int64 `json:"latencyMs"`
	Count     int64 `json:"count"`
}

type jsonResult struct {
	TotalRequests              int64            `json:"totalRequests"`
	TotalSuccessCodes          int64            `json:"totalSuccessCodes"`
	TotalErrors                int64            `json:"totalErrors"`
	TotalIDmismatch            int64            `json:"TotalIDmismatch"`
	TotalTruncatedResponses    int64            `json:"totalTruncatedResponses"`
	ResponseRcodes             map[string]int64 `json:"responseRcodes,omitempty"`
	QuestionTypes              map[string]int64 `json:"questionTypes"`
	QueriesPerSecond           float64          `json:"queriesPerSecond"`
	BenchmarkDurationSeconds   float64          `json:"benchmarkDurationSeconds"`
	LatencyStats               latencyStats     `json:"latencyStats"`
	LatencyDistribution        []histogramPoint `json:"latencyDistribution,omitempty"`
	TotalDNSSECSecuredDomains  *int             `json:"totalDNSSECSecuredDomains,omitempty"`
	DohHTTPResponseStatusCodes map[int]int64    `json:"dohHTTPResponseStatusCodes,omitempty"`
}

func (s *jsonReporter) print(params reportParameters) error {
	sumerrs := int64(0)
	for _, v := range params.topErrs.m {
		sumerrs += int64(v)
	}

	codeTotalsMapped := make(map[string]int64)
	if params.benchmark.Rcodes {
		for k, v := range params.codeTotals {
			codeTotalsMapped[dns.RcodeToString[k]] = v
		}
	}

	var res []histogramPoint

	if params.benchmark.HistDisplay {
		dist := params.timings.Distribution()
		for _, d := range dist {
			res = append(res, histogramPoint{
				LatencyMs: time.Duration(d.To/2 + d.From/2).Milliseconds(),
				Count:     d.Count,
			})
		}

		var dedupRes []histogramPoint
		i := -1
		for _, r := range res {
			if i >= 0 && i < len(res) {
				if dedupRes[i].LatencyMs == r.LatencyMs {
					dedupRes[i].Count += r.Count
				} else {
					dedupRes = append(dedupRes, r)
					i++
				}
			} else {
				dedupRes = append(dedupRes, r)
				i++
			}
		}
	}

	result := jsonResult{
		TotalRequests:            params.totalCounters.Total,
		TotalSuccessCodes:        params.totalCounters.Success,
		TotalErrors:              sumerrs,
		TotalIDmismatch:          params.totalCounters.IDmismatch,
		TotalTruncatedResponses:  params.totalCounters.Truncated,
		QueriesPerSecond:         math.Round(float64(params.totalCounters.Total)/params.benchmarkDuration.Seconds()*100) / 100,
		BenchmarkDurationSeconds: roundDuration(params.benchmarkDuration).Seconds(),
		ResponseRcodes:           codeTotalsMapped,
		QuestionTypes:            params.qtypeTotals,
		LatencyStats: latencyStats{
			MinMs:  time.Duration(params.timings.Min()).Milliseconds(),
			MeanMs: time.Duration(params.timings.Mean()).Milliseconds(),
			StdMs:  time.Duration(params.timings.StdDev()).Milliseconds(),
			MaxMs:  time.Duration(params.timings.Max()).Milliseconds(),
			P99Ms:  time.Duration(params.timings.ValueAtQuantile(99)).Milliseconds(),
			P95Ms:  time.Duration(params.timings.ValueAtQuantile(95)).Milliseconds(),
			P90Ms:  time.Duration(params.timings.ValueAtQuantile(90)).Milliseconds(),
			P75Ms:  time.Duration(params.timings.ValueAtQuantile(75)).Milliseconds(),
			P50Ms:  time.Duration(params.timings.ValueAtQuantile(50)).Milliseconds(),
		},
		LatencyDistribution:        res,
		DohHTTPResponseStatusCodes: params.dohResponseStatusesTotals,
	}
	if params.benchmark.DNSSEC {
		totalDNSSECSecuredDomains := len(params.authenticatedDomains)
		result.TotalDNSSECSecuredDomains = &totalDNSSECSecuredDomains
	}

	return json.NewEncoder(params.outputWriter).Encode(result)
}
