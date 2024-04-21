package dnsbench

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

var now = time.Now()

func TestResultStats_record(t *testing.T) {
	type args struct {
		req          *dns.Msg
		resp         *dns.Msg
		err          error
		time         time.Time
		duration     time.Duration
		dohBenchmark bool
	}
	tests := []struct {
		name string
		args args
		want *ResultStats
	}{
		{
			name: "record success",
			args: args{
				req: &dns.Msg{
					MsgHdr: dns.MsgHdr{Id: 1},
					Question: []dns.Question{
						{
							Name:   "example.org.",
							Qclass: dns.ClassINET,
							Qtype:  dns.TypeA,
						},
					},
				},
				resp: &dns.Msg{
					MsgHdr: dns.MsgHdr{Id: 1, Rcode: dns.RcodeSuccess, Response: true},
					Answer: []dns.RR{&dns.A{A: net.ParseIP("127.0.0.1")}},
				},
				time:     time.Now(),
				duration: time.Millisecond,
			},
			want: &ResultStats{
				Codes: map[int]int64{
					dns.RcodeSuccess: 1,
				},
				Qtypes: map[string]int64{
					"A": 1,
				},
				Timings: []Datapoint{
					{
						Duration: time.Millisecond,
						Start:    now,
					},
				},
				Counters: &Counters{
					Total:   1,
					Success: 1,
				},
			},
		},
		{
			name: "record nxdomain",
			args: args{
				req: &dns.Msg{
					MsgHdr: dns.MsgHdr{Id: 1},
					Question: []dns.Question{
						{
							Name:   "example.org.",
							Qclass: dns.ClassINET,
							Qtype:  dns.TypeA,
						},
					},
				},
				resp: &dns.Msg{
					MsgHdr: dns.MsgHdr{Id: 1, Rcode: dns.RcodeNameError, Response: true},
				},
				time:     time.Now(),
				duration: time.Millisecond,
			},
			want: &ResultStats{
				Codes: map[int]int64{
					dns.RcodeNameError: 1,
				},
				Qtypes: map[string]int64{
					"A": 1,
				},
				Timings: []Datapoint{
					{
						Duration: time.Millisecond,
						Start:    now,
					},
				},
				Counters: &Counters{
					Total:    1,
					Negative: 1,
				},
			},
		},
		{
			name: "record nodata",
			args: args{
				req: &dns.Msg{
					MsgHdr: dns.MsgHdr{Id: 1},
					Question: []dns.Question{
						{
							Name:   "example.org.",
							Qclass: dns.ClassINET,
							Qtype:  dns.TypeA,
						},
					},
				},
				resp: &dns.Msg{
					MsgHdr: dns.MsgHdr{Id: 1, Rcode: dns.RcodeSuccess, Response: true},
				},
				time:     time.Now(),
				duration: time.Millisecond,
			},
			want: &ResultStats{
				Codes: map[int]int64{
					dns.RcodeSuccess: 1,
				},
				Qtypes: map[string]int64{
					"A": 1,
				},
				Timings: []Datapoint{
					{
						Duration: time.Millisecond,
						Start:    now,
					},
				},
				Counters: &Counters{
					Total:    1,
					Negative: 1,
				},
			},
		},
		{
			name: "record dns error",
			args: args{
				req: &dns.Msg{
					MsgHdr: dns.MsgHdr{Id: 1},
					Question: []dns.Question{
						{
							Name:   "example.org.",
							Qclass: dns.ClassINET,
							Qtype:  dns.TypeA,
						},
					},
				},
				resp: &dns.Msg{
					MsgHdr: dns.MsgHdr{Id: 1, Rcode: dns.RcodeServerFailure, Response: true},
				},
				time:     time.Now(),
				duration: time.Millisecond,
			},
			want: &ResultStats{
				Codes: map[int]int64{
					dns.RcodeServerFailure: 1,
				},
				Qtypes: map[string]int64{
					"A": 1,
				},
				Timings: []Datapoint{
					{
						Duration: time.Millisecond,
						Start:    now,
					},
				},
				Counters: &Counters{
					Total: 1,
					Error: 1,
				},
			},
		},
		{
			name: "record IO error",
			args: args{
				req: &dns.Msg{
					MsgHdr: dns.MsgHdr{Id: 1},
					Question: []dns.Question{
						{
							Name:   "example.org.",
							Qclass: dns.ClassINET,
							Qtype:  dns.TypeA,
						},
					},
				},
				err:      errors.New("test error"),
				time:     time.Now(),
				duration: time.Millisecond,
			},
			want: &ResultStats{
				Codes: map[int]int64{},
				Qtypes: map[string]int64{
					"A": 1,
				},
				Errors: []ErrorDatapoint{
					{
						Err:   errors.New("test error"),
						Start: now,
					},
				},
				Counters: &Counters{
					Total:   1,
					IOError: 1,
				},
			},
		},
		{
			name: "record truncated response",
			args: args{
				req: &dns.Msg{
					MsgHdr: dns.MsgHdr{Id: 1},
					Question: []dns.Question{
						{
							Name:   "example.org.",
							Qclass: dns.ClassINET,
							Qtype:  dns.TypeA,
						},
					},
				},
				resp: &dns.Msg{
					MsgHdr: dns.MsgHdr{Id: 1, Rcode: dns.RcodeSuccess, Response: true, Truncated: true},
					Answer: []dns.RR{&dns.A{A: net.ParseIP("127.0.0.1")}},
				},
				time:     time.Now(),
				duration: time.Millisecond,
			},
			want: &ResultStats{
				Codes: map[int]int64{
					dns.RcodeSuccess: 1,
				},
				Qtypes: map[string]int64{
					"A": 1,
				},
				Timings: []Datapoint{
					{
						Duration: time.Millisecond,
						Start:    now,
					},
				},
				Counters: &Counters{
					Total:     1,
					Truncated: 1,
					Success:   1,
				},
			},
		},
		{
			name: "record response ID mismatch",
			args: args{
				req: &dns.Msg{
					MsgHdr: dns.MsgHdr{Id: 1},
					Question: []dns.Question{
						{
							Name:   "example.org.",
							Qclass: dns.ClassINET,
							Qtype:  dns.TypeA,
						},
					},
				},
				resp: &dns.Msg{
					MsgHdr: dns.MsgHdr{Id: 2, Rcode: dns.RcodeSuccess, Response: true},
					Answer: []dns.RR{&dns.A{A: net.ParseIP("127.0.0.1")}},
				},
				time:     time.Now(),
				duration: time.Millisecond,
			},
			want: &ResultStats{
				Codes: map[int]int64{},
				Qtypes: map[string]int64{
					"A": 1,
				},
				Counters: &Counters{
					Total:      1,
					IDmismatch: 1,
				},
			},
		},
		{
			name: "record DoH success",
			args: args{
				req: &dns.Msg{
					MsgHdr: dns.MsgHdr{Id: 1},
					Question: []dns.Question{
						{
							Name:   "example.org.",
							Qclass: dns.ClassINET,
							Qtype:  dns.TypeA,
						},
					},
				},
				resp: &dns.Msg{
					MsgHdr: dns.MsgHdr{Id: 1, Rcode: dns.RcodeSuccess, Response: true},
					Answer: []dns.RR{&dns.A{A: net.ParseIP("127.0.0.1")}},
				},
				time:         time.Now(),
				duration:     time.Millisecond,
				dohBenchmark: true,
			},
			want: &ResultStats{
				Codes: map[int]int64{
					dns.RcodeSuccess: 1,
				},
				Qtypes: map[string]int64{
					"A": 1,
				},
				Timings: []Datapoint{
					{
						Duration: time.Millisecond,
						Start:    now,
					},
				},
				Counters: &Counters{
					Total:   1,
					Success: 1,
				},
				DoHStatusCodes: map[int]int64{
					200: 1,
				},
			},
		},
		{
			name: "record authenticated domain",
			args: args{
				req: &dns.Msg{
					MsgHdr: dns.MsgHdr{Id: 1},
					Question: []dns.Question{
						{
							Name:   "example.org.",
							Qclass: dns.ClassINET,
							Qtype:  dns.TypeA,
						},
					},
				},
				resp: &dns.Msg{
					MsgHdr: dns.MsgHdr{Id: 1, Rcode: dns.RcodeSuccess, Response: true, AuthenticatedData: true},
					Answer: []dns.RR{&dns.A{A: net.ParseIP("127.0.0.1")}},
				},
				time:     time.Now(),
				duration: time.Millisecond,
			},
			want: &ResultStats{
				Codes: map[int]int64{
					dns.RcodeSuccess: 1,
				},
				Qtypes: map[string]int64{
					"A": 1,
				},
				Timings: []Datapoint{
					{
						Duration: time.Millisecond,
						Start:    now,
					},
				},
				Counters: &Counters{
					Total:   1,
					Success: 1,
				},
				AuthenticatedDomains: map[string]struct{}{
					"example.org.": {},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := Benchmark{
				Rcodes: true,
				useDoH: tt.args.dohBenchmark,
			}
			rs := newResultStats(&b)

			rs.record(tt.args.req, tt.args.resp, tt.args.err, now, tt.args.duration)

			// null the Histogram for simple assertion excluding the histogram
			rs.Hist = nil

			assert.Equal(t, tt.want, rs)
		})
	}
}
