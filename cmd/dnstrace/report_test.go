package dnstrace

import (
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/miekg/dns"
)

func Example_printReport() {
	pExpect = &[]string{"A"}

	h := hdrhistogram.New(pHistMin.Nanoseconds(), pHistMax.Nanoseconds(), 1)
	h.RecordValue(5)
	h.RecordValue(10)
	d1 := datapoint{5, time.Unix(0, 0)}
	d2 := datapoint{10, time.Unix(0, 0)}
	rs := rstats{
		codes: map[int]int64{
			dns.RcodeSuccess: 2,
		},
		qtypes: map[string]int64{
			"A": 2,
		},
		hist:      h,
		timings:   []datapoint{d1, d2},
		count:     1,
		cerror:    2,
		ecount:    3,
		success:   4,
		matched:   5,
		mismatch:  6,
		truncated: 7,
	}

	printReport(time.Second, []*rstats{&rs}, nil)

	//Output: Total requests:		1
	//Connection errors:	2
	//Read/Write errors:	3
	//ID mismatch errors:	6
	//DNS success codes:	4
	//Truncated responses:	7
	//Expected results:	5
	//
	//DNS response codes:
	//	NOERROR:	2
	//
	//DNS question types:
	//	A:	2
	//
	//Time taken for tests:	 1s
	//Questions per second:	 1.0
	//DNS timings, 2 datapoints
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
