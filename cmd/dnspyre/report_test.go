package dnspyre

import (
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/miekg/dns"
)

func Example_printReport() {
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
			ConnError:  2,
			IOError:    3,
			Success:    4,
			IDmismatch: 6,
			Truncated:  7,
		},
	}

	b.PrintReport([]*ResultStats{&rs}, time.Second)

	// Output: Total requests:		1
	// Connection errors:	2
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
}
