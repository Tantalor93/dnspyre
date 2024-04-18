package cmd

import (
	"errors"
	"net"
	"os"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/miekg/dns"
)

func ExampleBenchmark_PrintReport() {
	b, rs := testReportData()

	b.PrintReport(os.Stdout, []*ResultStats{&rs}, time.Now(), time.Second)

	// Output: Total requests:		1
	// Read/Write errors:	6
	// ID mismatch errors:	10
	// DNS success responses:	4
	// DNS negative responses:	8
	// DNS error responses:	9
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
	// Total Errors: 6
	// Top errors:
	// test2	3 (50.00)%
	// read udp 8.8.8.8:53	2 (33.33)%
	// test	1 (16.67)%
}

func ExampleBenchmark_PrintReport_dnssec() {
	b, rs := testReportData()
	b.DNSSEC = true
	rs.AuthenticatedDomains = map[string]struct{}{"example.org.": {}}

	b.PrintReport(os.Stdout, []*ResultStats{&rs}, time.Now(), time.Second)

	// Output: Total requests:		1
	// Read/Write errors:	6
	// ID mismatch errors:	10
	// DNS success responses:	4
	// DNS negative responses:	8
	// DNS error responses:	9
	// Truncated responses:	7
	//
	// DNS response codes:
	//	NOERROR:	2
	//
	// DNS question types:
	//	A:	2
	//
	// Number of domains secured using DNSSEC: 1
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
	// Total Errors: 6
	// Top errors:
	// test2	3 (50.00)%
	// read udp 8.8.8.8:53	2 (33.33)%
	// test	1 (16.67)%
}

func ExampleBenchmark_PrintReport_doh() {
	b, rs := testReportData()
	rs.DoHStatusCodes = map[int]int64{
		200: 2,
		500: 1,
	}

	b.PrintReport(os.Stdout, []*ResultStats{&rs}, time.Now(), time.Second)

	// Output: Total requests:		1
	// Read/Write errors:	6
	// ID mismatch errors:	10
	// DNS success responses:	4
	// DNS negative responses:	8
	// DNS error responses:	9
	// Truncated responses:	7
	//
	// DNS response codes:
	//	NOERROR:	2
	//
	// DoH HTTP response status codes:
	//	200:	2
	//	500:	1
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
	// Total Errors: 6
	// Top errors:
	// test2	3 (50.00)%
	// read udp 8.8.8.8:53	2 (33.33)%
	// test	1 (16.67)%
}

func ExampleBenchmark_PrintReport_json() {
	b, rs := testReportData()
	b.JSON = true
	b.Rcodes = true
	b.HistDisplay = true

	b.PrintReport(os.Stdout, []*ResultStats{&rs}, time.Now(), time.Second)

	// Output: {"totalRequests":1,"totalSuccessResponses":4,"totalNegativeResponses":8,"totalErrorResponses":9,"totalIOErrors":6,"TotalIDmismatch":10,"totalTruncatedResponses":7,"responseRcodes":{"NOERROR":2},"questionTypes":{"A":2},"queriesPerSecond":1,"benchmarkDurationSeconds":1,"latencyStats":{"minMs":0,"meanMs":0,"stdMs":0,"maxMs":0,"p99Ms":0,"p95Ms":0,"p90Ms":0,"p75Ms":0,"p50Ms":0},"latencyDistribution":[{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":1},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":1}]}
}

func ExampleBenchmark_PrintReport_json_dnssec() {
	b, rs := testReportData()
	b.JSON = true
	b.Rcodes = true
	b.HistDisplay = true
	b.DNSSEC = true
	rs.AuthenticatedDomains = map[string]struct{}{"example.org.": {}}

	b.PrintReport(os.Stdout, []*ResultStats{&rs}, time.Now(), time.Second)

	// Output: {"totalRequests":1,"totalSuccessResponses":4,"totalNegativeResponses":8,"totalErrorResponses":9,"totalIOErrors":6,"TotalIDmismatch":10,"totalTruncatedResponses":7,"responseRcodes":{"NOERROR":2},"questionTypes":{"A":2},"queriesPerSecond":1,"benchmarkDurationSeconds":1,"latencyStats":{"minMs":0,"meanMs":0,"stdMs":0,"maxMs":0,"p99Ms":0,"p95Ms":0,"p90Ms":0,"p75Ms":0,"p50Ms":0},"latencyDistribution":[{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":1},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":1}],"totalDNSSECSecuredDomains":1}
}

func ExampleBenchmark_PrintReport_json_doh() {
	b, rs := testReportData()
	b.JSON = true
	b.Rcodes = true
	b.HistDisplay = true
	rs.DoHStatusCodes = map[int]int64{
		200: 2,
	}

	b.PrintReport(os.Stdout, []*ResultStats{&rs}, time.Now(), time.Second)

	// Output: {"totalRequests":1,"totalSuccessResponses":4,"totalNegativeResponses":8,"totalErrorResponses":9,"totalIOErrors":6,"TotalIDmismatch":10,"totalTruncatedResponses":7,"responseRcodes":{"NOERROR":2},"questionTypes":{"A":2},"queriesPerSecond":1,"benchmarkDurationSeconds":1,"latencyStats":{"minMs":0,"meanMs":0,"stdMs":0,"maxMs":0,"p99Ms":0,"p95Ms":0,"p90Ms":0,"p75Ms":0,"p50Ms":0},"latencyDistribution":[{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":1},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":0},{"latencyMs":0,"count":1}],"dohHTTPResponseStatusCodes":{"200":2}}
}

func ExampleBenchmark_PrintReport_server_dns_errors() {
	b, rs := testReportDataWithServerDNSErrors()

	b.PrintReport(os.Stdout, []*ResultStats{&rs}, time.Now(), time.Second)

	// Output: Total requests:		3
	// Read/Write errors:	3
	//
	// DNS question types:
	// 	A:	3
	//
	// Time taken for tests:	 1s
	// Questions per second:	 3.0
	//
	// Total Errors: 3
	// Top errors:
	// no such host unknown.host.com	3 (100.00)%
}

func testReportData() (Benchmark, ResultStats) {
	b := Benchmark{
		HistPre: 1,
	}

	h := hdrhistogram.New(0, 0, 1)
	h.RecordValue(5)
	h.RecordValue(10)
	d1 := Datapoint{5, time.Unix(0, 0)}
	d2 := Datapoint{10, time.Unix(0, 0)}
	addr, err := net.ResolveUDPAddr("udp", "8.8.8.8:53")
	if err != nil {
		panic(err)
	}
	saddr1, err := net.ResolveUDPAddr("udp", "127.0.0.1:65359")
	if err != nil {
		panic(err)
	}
	saddr2, err := net.ResolveUDPAddr("udp", "127.0.0.1:65360")
	if err != nil {
		panic(err)
	}
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
			IOError:    6,
			Success:    4,
			IDmismatch: 10,
			Truncated:  7,
			Negative:   8,
			Error:      9,
		},
		Errors: []ErrorDatapoint{
			{Start: time.Unix(0, 0), Err: errors.New("test2")},
			{Start: time.Unix(0, 0), Err: errors.New("test")},
			{Start: time.Unix(0, 0), Err: &net.OpError{Op: "read", Net: udpNetwork, Addr: addr, Source: saddr1}},
			{Start: time.Unix(0, 0), Err: &net.OpError{Op: "read", Net: udpNetwork, Addr: addr, Source: saddr2}},
			{Start: time.Unix(0, 0), Err: errors.New("test2")},
			{Start: time.Unix(0, 0), Err: errors.New("test2")},
		},
	}
	return b, rs
}

func testReportDataWithServerDNSErrors() (Benchmark, ResultStats) {
	b := Benchmark{
		HistPre: 1,
	}
	h := hdrhistogram.New(0, 0, 1)
	rs := ResultStats{
		Codes: map[int]int64{},
		Qtypes: map[string]int64{
			"A": 3,
		},
		Hist:    h,
		Timings: []Datapoint{},
		Counters: &Counters{
			Total:   3,
			IOError: 3,
		},
		Errors: []ErrorDatapoint{
			{Start: time.Unix(0, 0), Err: &net.DNSError{Err: "no such host", Name: "unknown.host.com"}},
			{Start: time.Unix(0, 0), Err: &net.DNSError{Err: "no such host", Name: "unknown.host.com"}},
			{Start: time.Unix(0, 0), Err: &net.DNSError{Err: "no such host", Name: "unknown.host.com"}},
		},
	}
	return b, rs
}
