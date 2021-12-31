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
	"github.com/stretchr/testify/require"
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

			setupBenchmarkTest(s.Addr, tt.args.protocol == tcp)
			resetPackageCounters()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			rs := do(ctx)

			assertResult(t, rs)
		})
	}
}

func Test_do_doh_post(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bd, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err, "error reading body")

		msg := dns.Msg{}
		err = msg.Unpack(bd)
		require.NoError(t, err, "error unpacking request body")
		require.Len(t, msg.Question, 1, "single question expected")

		msg.Answer = append(msg.Answer, test.A("example.org. IN A 127.0.0.1"))

		pack, err := msg.Pack()
		require.NoError(t, err, "error packing response")

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		_, err = w.Write(pack)
		require.NoError(t, err, "error writing response")
	}))
	defer ts.Close()

	*pDoHmethod = post

	setupBenchmarkTest(ts.URL, true)
	resetPackageCounters()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs := do(ctx)

	assertResult(t, rs)
}

func Test_do_doh_get(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		dnsQryParam := query.Get("dns")
		require.NotEmpty(t, dnsQryParam, "expected dns query param not found")

		bd, err := base64.StdEncoding.DecodeString(dnsQryParam)
		require.NoError(t, err, "error decoding query param DNS")

		msg := dns.Msg{}
		err = msg.Unpack(bd)
		require.NoError(t, err, "error unpacking request body")
		require.Len(t, msg.Question, 1, "single question expected")

		msg.Answer = append(msg.Answer, test.A("example.org. IN A 127.0.0.1"))

		pack, err := msg.Pack()
		require.NoError(t, err, "error packing response")

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		_, err = w.Write(pack)
		require.NoError(t, err, "error writing response")
	}))
	defer ts.Close()

	*pDoHmethod = get

	setupBenchmarkTest(ts.URL, true)
	resetPackageCounters()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs := do(ctx)

	assertResult(t, rs)
}

func assertResult(t *testing.T, rs []*rstats) {
	if assert.Len(t, rs, 2, "do(ctx) rstats") {
		if assert.NotNil(t, rs[0].hist, "do(ctx) rstats histogram") {
			assert.NotNil(t, rs[0].codes, "do(ctx) rstats codes")
			assert.Equal(t, int64(1), rs[0].codes[0], "do(ctx) rstats codes NOERROR, state:"+fmt.Sprint(rs[0].codes))
		}

		if assert.NotNil(t, rs[1].hist, "do(ctx) rstats histogram") {
			assert.NotNil(t, rs[1].codes, "do(ctx) rstats codes")
			assert.Equal(t, int64(1), rs[1].codes[0], "do(ctx) rstats codes NOERROR, state:"+fmt.Sprint(rs[1].codes))
		}

		if assert.Len(t, rs[0].timings, 1, "do(ctx) rstats timings") {
			assert.NotZero(t, rs[0].timings[0].duration, "do(ctx) rstats timings duration")
			assert.NotZero(t, rs[0].timings[0].start, "do(ctx) rstats timings start")
		}

		if assert.Len(t, rs[1].timings, 1, "do(ctx) rstats timings") {
			assert.NotZero(t, rs[1].timings[0].duration, "do(ctx) rstats timings duration")
			assert.NotZero(t, rs[1].timings[0].start, "do(ctx) rstats timings start")
		}
	}

	assert.Equal(t, int64(2), count, "total counter")
	assert.Zero(t, cerror, "connection error counter")
	assert.Zero(t, ecount, "error counter")
	assert.Equal(t, int64(2), success, "success counter")
	assert.Equal(t, int64(2), matched, "matched counter")
	assert.Zero(t, mismatch, "mismatch counter")
	assert.Zero(t, truncated, "truncated counter")
}

func setupBenchmarkTest(server string, tcp bool) {
	pQueries = &[]string{"example.org."}

	typ := "A"
	pType = &typ

	pServer = &server
	pTCP = &tcp

	concurrency := uint32(2)
	pConcurrency = &concurrency

	c := int64(1)
	pCount = &c

	probability := float64(1)
	pProbability = &probability

	writeTimeout := 5 * time.Second
	pWriteTimeout = &writeTimeout

	readTimeout := 5 * time.Second
	pReadTimeout = &readTimeout

	rcodes := true
	pRCodes = &rcodes

	expect := []string{"A"}
	pExpect = &expect

	recurse := true
	pRecurse = &recurse
}
