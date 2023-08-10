package cmd

import (
	"errors"
	"os"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/miekg/dns"
)

func Example_standard_printReport() {
	b, rs := testData()

	b.PrintReport(os.Stdout, []*ResultStats{&rs}, time.Second)

	// Output: Total requests:		1
	// Read/Write errors:	3
	// ID mismatch errors:	6
	// DNS success codes:	4
	// Truncated responses:	7
	//
	// DNS response codes:
	//	NOERROR:	2
	//
	// DNS question types:
	//	A:	2
	//
	// Time taken for tests:	 1s
	// Questions per second:	 1.0
	// DNS timings, 2 datapoints
	//	 min:		 5ns
	//	 mean:		 7ns
	//	 [+/-sd]:	 2ns
	//	 max:		 10ns
	//	 p99:		 10ns
	//	 p95:		 10ns
	//	 p90:		 10ns
	//	 p75:		 10ns
	//	 p50:		 5ns
	//
	// Total Errors: 3
	// Top errors:
	// test	2 (66.67)%
	// test2	1 (33.33)%
}

func Example_json_printReport() {
	b, rs := testData()
	b.JSON = true
	b.Rcodes = true
	b.HistDisplay = true

	b.PrintReport(os.Stdout, []*ResultStats{&rs}, time.Second)

	// Output: {"totalRequests":1,"totalSuccessCodes":4,"totalErrors":3,"TotalIDmismatch":6,"totalTruncatedResponses":7,"responseRcodes":{"NOERROR":2},"questionTypes":{"A":2},"queriesPerSecond":1,"benchmarkDurationSeconds":1,"latencyStats":{"minMs":0,"meanMs":0,"stdMs":0,"maxMs":0,"p99Ms":0,"p95Ms":0,"p90Ms":0,"p75Ms":0,"p50Ms":0},"latencyDistribution":[{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":1},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":1}]}
}

func testData() (Benchmark, ResultStats) {
	b := Benchmark{
		HistPre: 1,
	}

	h := hdrhistogram.New(0, 0, 1)
	h.RecordValue(5)
	h.RecordValue(10)
	d1 := Datapoint{5, time.Unix(0, 0)}
	d2 := Datapoint{10, time.Unix(0, 0)}
	rs := ResultStats{
		Codes: map[int]int64{
			dns.RcodeSuccess: 2,
		},
		Qtypes: map[string]int64{
			"A": 2,
		},
		Hist:    h,
		Timings: []Datapoint{d1, d2},
		Counters: &Counters{
			Total:      1,
			IOError:    3,
			Success:    4,
			IDmismatch: 6,
			Truncated:  7,
		},
		Errors: []error{
			errors.New("test"),
			errors.New("test2"),
			errors.New("test"),
		},
	}
	return b, rs
}
