package reporter

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
	TotalSuccessResponses      int64            `json:"totalSuccessResponses"`
	TotalNegativeResponses     int64            `json:"totalNegativeResponses"`
	TotalErrorResponses        int64            `json:"totalErrorResponses"`
	TotalIOErrors              int64            `json:"totalIOErrors"`
	TotalIDmismatch            int64            `json:"totalIDmismatch"`
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
	codeTotalsMapped := make(map[string]int64)
	if params.benchmark.Rcodes {
		for k, v := range params.codeTotals {
			codeTotalsMapped[dns.RcodeToString[k]] = v
		}
	}

	var res []histogramPoint

	if params.benchmark.HistDisplay {
		dist := params.hist.Distribution()
		for _, d := range dist {
			res = append(res, histogramPoint{
				LatencyMs: roundDuration(time.Duration(d.To/2 + d.From/2)).Milliseconds(),
				Count:     d.Count,
			})
		}

		var dedupRes []histogramPoint
		i := -1
		for _, r := range res {
			if i >= 0 {
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
		res = dedupRes
	}

	result := jsonResult{
		TotalRequests:            params.totalCounters.Total,
		TotalSuccessResponses:    params.totalCounters.Success,
		TotalNegativeResponses:   params.totalCounters.Negative,
		TotalErrorResponses:      params.totalCounters.Error,
		TotalIOErrors:            params.totalCounters.IOError,
		TotalIDmismatch:          params.totalCounters.IDmismatch,
		TotalTruncatedResponses:  params.totalCounters.Truncated,
		QueriesPerSecond:         math.Round(float64(params.totalCounters.Total)/params.benchmarkDuration.Seconds()*100) / 100,
		BenchmarkDurationSeconds: roundDuration(params.benchmarkDuration).Seconds(),
		ResponseRcodes:           codeTotalsMapped,
		QuestionTypes:            params.qtypeTotals,
		LatencyStats: latencyStats{
			MinMs:  roundDuration(time.Duration(params.hist.Min())).Milliseconds(),
			MeanMs: roundDuration(time.Duration(params.hist.Mean())).Milliseconds(),
			StdMs:  roundDuration(time.Duration(params.hist.StdDev())).Milliseconds(),
			MaxMs:  roundDuration(time.Duration(params.hist.Max())).Milliseconds(),
			P99Ms:  roundDuration(time.Duration(params.hist.ValueAtQuantile(99))).Milliseconds(),
			P95Ms:  roundDuration(time.Duration(params.hist.ValueAtQuantile(95))).Milliseconds(),
			P90Ms:  roundDuration(time.Duration(params.hist.ValueAtQuantile(90))).Milliseconds(),
			P75Ms:  roundDuration(time.Duration(params.hist.ValueAtQuantile(75))).Milliseconds(),
			P50Ms:  roundDuration(time.Duration(params.hist.ValueAtQuantile(50))).Milliseconds(),
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
