package dnspyre

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

const (
	udp = "udp"
	tcp = "tcp"

	get  = "get"
	post = "post"
)

func Test_do_classic_dns(t *testing.T) {
	type args struct {
		protocol string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"benchmark - DNS over UDP",
			args{
				protocol: udp,
			},
		},
		{
			"benchmark - DNS over TCP",
			args{
				protocol: tcp,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(tt.args.protocol, func(w dns.ResponseWriter, r *dns.Msg) {
				ret := new(dns.Msg)
				ret.SetReply(r)
				ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

				// wait some time to actually have some observable duration
				time.Sleep(time.Millisecond * 500)

				w.WriteMsg(ret)
			})
			defer s.Close()

			bench := createBenchmark(s.Addr, tt.args.protocol == tcp, 1)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			rs := bench.Run(ctx)

			assertResult(t, rs)
		})
	}
}

func Test_do_doh_post(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bd, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}

		msg := dns.Msg{}
		err = msg.Unpack(bd)
		if err != nil {
			panic(err)
		}

		msg.Answer = append(msg.Answer, A("example.org. IN A 127.0.0.1"))

		pack, err := msg.Pack()
		if err != nil {
			panic(err)
		}

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		_, err = w.Write(pack)
		if err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	bench := createBenchmark(ts.URL, true, 1)
	bench.DohMethod = post

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs := bench.Run(ctx)

	assertResult(t, rs)
}

func Test_do_doh_get(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		dnsQryParam := query.Get("dns")
		bd, err := base64.RawURLEncoding.DecodeString(dnsQryParam)
		if err != nil {
			panic(err)
		}

		msg := dns.Msg{}
		err = msg.Unpack(bd)
		if err != nil {
			panic(err)
		}

		msg.Answer = append(msg.Answer, A("example.org. IN A 127.0.0.1"))

		pack, err := msg.Pack()
		if err != nil {
			panic(err)
		}

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		_, err = w.Write(pack)
		if err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	bench := createBenchmark(ts.URL, true, 1)
	bench.DohMethod = get

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs := bench.Run(ctx)

	assertResult(t, rs)
}

func Test_do_probability(t *testing.T) {
	s := NewServer(udp, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		w.WriteMsg(ret)
	})
	defer s.Close()

	bench := createBenchmark(s.Addr, false, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs := bench.Run(ctx)

	assert.Len(t, rs, 2, "Run(ctx) rstats")
	rs0 := rs[0]
	rs1 := rs[1]
	assert.Equal(t, int64(0), rs0.Counters.Total, "Run(ctx) total counter")
	assert.Equal(t, int64(0), rs1.Counters.Total, "Run(ctx) total counter")
}

func Test_download_external_datasource_using_http(t *testing.T) {
	s := NewServer("udp", func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		w.WriteMsg(ret)
	})
	defer s.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`example.org`))
		if err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	bench := Benchmark{
		Queries:      []string{ts.URL},
		Types:        []string{"A", "AAAA"},
		Server:       s.Addr,
		TCP:          false,
		Concurrency:  2,
		Count:        1,
		Probability:  1,
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
		Rcodes:       true,
		Recurse:      true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs := bench.Run(ctx)

	assertResult(t, rs)
}

func Test_do_classic_dns_with_duration(t *testing.T) {
	s := NewServer("udp", func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))
		w.WriteMsg(ret)
	})
	defer s.Close()

	bench := Benchmark{
		Queries:      []string{"example.org"},
		Types:        []string{"A"},
		Server:       s.Addr,
		Concurrency:  1,
		Duration:     2 * time.Second,
		Probability:  1,
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
		Rcodes:       true,
		Recurse:      true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs := bench.Run(ctx)

	assert.GreaterOrEqual(t, rs[0].Counters.Total, int64(1), "there should be atleast one execution")
}

func assertResult(t *testing.T, rs []*ResultStats) {
	if assert.Len(t, rs, 2, "Run(ctx) rstats") {
		rs0 := rs[0]
		rs1 := rs[1]
		assertResultStats(t, rs0)
		assertResultStats(t, rs1)
		assertTimings(t, rs0)
		assertTimings(t, rs1)
	}
}

func assertResultStats(t *testing.T, rs *ResultStats) {
	assert.NotNil(t, rs.Hist, "Run(ctx) rstats histogram")

	if assert.NotNil(t, rs.Codes, "Run(ctx) rstats codes") {
		assert.Equal(t, int64(2), rs.Codes[0], "Run(ctx) rstats codes NOERROR, state:"+fmt.Sprint(rs.Codes))
	}

	if assert.NotNil(t, rs.Qtypes, "Run(ctx) rstats qtypes") {
		assert.Equal(t, int64(1), rs.Qtypes[dns.TypeToString[dns.TypeA]], "Run(ctx) rstats qtypes A, state:"+fmt.Sprint(rs.Codes))
		assert.Equal(t, int64(1), rs.Qtypes[dns.TypeToString[dns.TypeAAAA]], "Run(ctx) rstats qtypes AAAA, state:"+fmt.Sprint(rs.Codes))
	}

	assert.Equal(t, int64(2), rs.Counters.Total, "Run(ctx) total counter")
	assert.Zero(t, rs.Counters.ConnError, "Run(ctx) connection error counter")
	assert.Zero(t, rs.Counters.IOError, "error counter")
	assert.Equal(t, int64(2), rs.Counters.Success, "Run(ctx) success counter")
	assert.Zero(t, rs.Counters.IDmismatch, "Run(ctx) mismatch counter")
	assert.Zero(t, rs.Counters.Truncated, "Run(ctx) truncated counter")
}

func assertTimings(t *testing.T, rs *ResultStats) {
	if assert.Len(t, rs.Timings, 2, "Run(ctx) rstats timings") {
		t0 := rs.Timings[0]
		t1 := rs.Timings[1]
		assert.NotZero(t, t0.Duration, "Run(ctx) rstats timings duration")
		assert.NotZero(t, t0.Start, "Run(ctx) rstats timings start")
		assert.NotZero(t, t1.Duration, "Run(ctx) rstats timings duration")
		assert.NotZero(t, t1.Start, "Run(ctx) rstats timings start")
	}
}

func createBenchmark(server string, tcp bool, prob float64) Benchmark {
	return Benchmark{
		Queries:      []string{"example.org"},
		Types:        []string{"A", "AAAA"},
		Server:       server,
		TCP:          tcp,
		Concurrency:  2,
		Count:        1,
		Probability:  prob,
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
		Rcodes:       true,
		Recurse:      true,
	}
}

// A returns an A record from rr. It panics on errors.
func A(rr string) *dns.A { r, _ := dns.NewRR(rr); return r.(*dns.A) }
