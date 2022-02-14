package dnstrace

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coredns/coredns/plugin/test"
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
				ret.Answer = append(ret.Answer, test.A("example.org. IN A 127.0.0.1"))

				// wait some time to actually have some observable duration
				time.Sleep(time.Millisecond * 500)

				w.WriteMsg(ret)
			})
			defer s.Close()

			benchmarkInput := prepareInput(s.Addr, tt.args.protocol == tcp)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			rs := do(ctx, benchmarkInput)

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

		msg.Answer = append(msg.Answer, test.A("example.org. IN A 127.0.0.1"))

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

	benchmarkInput := prepareInput(ts.URL, true)
	benchmarkInput.dohMethod = post

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs := do(ctx, benchmarkInput)

	assertResult(t, rs)
}

func Test_do_doh_get(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		dnsQryParam := query.Get("dns")
		bd, err := base64.URLEncoding.DecodeString(dnsQryParam)
		if err != nil {
			panic(err)
		}

		msg := dns.Msg{}
		err = msg.Unpack(bd)
		if err != nil {
			panic(err)
		}

		msg.Answer = append(msg.Answer, test.A("example.org. IN A 127.0.0.1"))

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

	benchmarkInput := prepareInput(ts.URL, true)
	benchmarkInput.dohMethod = get

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs := do(ctx, benchmarkInput)

	assertResult(t, rs)
}

func assertResult(t *testing.T, rs []*rstats) {
	if assert.Len(t, rs, 2, "do(ctx) rstats") {
		rs0 := rs[0]
		rs1 := rs[1]
		assertRstats(t, rs0)
		assertRstats(t, rs1)
		assertTimings(t, rs0)
		assertTimings(t, rs1)
	}
}

func assertRstats(t *testing.T, rs *rstats) {
	assert.NotNil(t, rs.hist, "do(ctx) rstats histogram")

	if assert.NotNil(t, rs.codes, "do(ctx) rstats codes") {
		assert.Equal(t, int64(2), rs.codes[0], "do(ctx) rstats codes NOERROR, state:"+fmt.Sprint(rs.codes))
	}

	if assert.NotNil(t, rs.qtypes, "do(ctx) rstats qtypes") {
		assert.Equal(t, int64(1), rs.qtypes[dns.TypeToString[dns.TypeA]], "do(ctx) rstats qtypes A, state:"+fmt.Sprint(rs.codes))
		assert.Equal(t, int64(1), rs.qtypes[dns.TypeToString[dns.TypeAAAA]], "do(ctx) rstats qtypes AAAA, state:"+fmt.Sprint(rs.codes))
	}

	assert.Equal(t, int64(2), rs.count, "do(ctx) total counter")
	assert.Zero(t, rs.cerror, "do(ctx) connection error counter")
	assert.Zero(t, rs.ecount, "error counter")
	assert.Equal(t, int64(2), rs.success, "do(ctx) success counter")
	assert.Equal(t, int64(2), rs.matched, "do(ctx) matched counter")
	assert.Zero(t, rs.mismatch, "do(ctx) mismatch counter")
	assert.Zero(t, rs.truncated, "do(ctx) truncated counter")
}

func assertTimings(t *testing.T, rs *rstats) {
	if assert.Len(t, rs.timings, 2, "do(ctx) rstats timings") {
		t0 := rs.timings[0]
		t1 := rs.timings[1]
		assert.NotZero(t, t0.duration, "do(ctx) rstats timings duration")
		assert.NotZero(t, t0.start, "do(ctx) rstats timings start")
		assert.NotZero(t, t1.duration, "do(ctx) rstats timings duration")
		assert.NotZero(t, t1.start, "do(ctx) rstats timings start")
	}
}

func prepareInput(server string, tcp bool) BenchmarkInput {
	return BenchmarkInput{
		queries:      []string{"example.org."},
		types:        []string{"A", "AAAA"},
		server:       server,
		tcp:          tcp,
		concurrency:  2,
		count:        1,
		probability:  1,
		writeTimeout: 5 * time.Second,
		readTimeout:  5 * time.Second,
		rcodes:       true,
		expect:       []string{"A"},
		recurse:      true,
	}
}
