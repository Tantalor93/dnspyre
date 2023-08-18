package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

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
			s := NewServer(tt.args.protocol, nil, func(w dns.ResponseWriter, r *dns.Msg) {
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
			rs, err := bench.Run(ctx)

			assert.NoError(t, err, "expected no error from benchmark run")
			assertResult(t, rs)
		})
	}
}

func Test_do_doh_post(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bd, err := io.ReadAll(r.Body)
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
	rs, err := bench.Run(ctx)

	assert.NoError(t, err, "expected no error from benchmark run")
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
	rs, err := bench.Run(ctx)

	assert.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
}

func Test_do_probability(t *testing.T) {
	s := NewServer(udp, nil, func(w dns.ResponseWriter, r *dns.Msg) {
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
	rs, err := bench.Run(ctx)

	assert.NoError(t, err, "expected no error from benchmark run")
	assert.Len(t, rs, 2, "Run(ctx) rstats")
	rs0 := rs[0]
	rs1 := rs[1]
	assert.Equal(t, int64(0), rs0.Counters.Total, "Run(ctx) total counter")
	assert.Equal(t, int64(0), rs1.Counters.Total, "Run(ctx) total counter")
}

func Test_download_external_datasource_using_http(t *testing.T) {
	s := NewServer("udp", nil, func(w dns.ResponseWriter, r *dns.Msg) {
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
		Queries:        []string{ts.URL},
		Types:          []string{"A", "AAAA"},
		Server:         s.Addr,
		TCP:            false,
		Concurrency:    2,
		Count:          1,
		Probability:    1,
		WriteTimeout:   1 * time.Second,
		ReadTimeout:    3 * time.Second,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
		Rcodes:         true,
		Recurse:        true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	assert.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
}

func Test_download_external_datasource_using_http_not_available(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	// close right away to get dead port
	ts.Close()

	bench := Benchmark{
		Queries:        []string{ts.URL},
		Types:          []string{"A", "AAAA"},
		Server:         "8.8.8.8",
		TCP:            false,
		Concurrency:    2,
		Count:          1,
		Probability:    1,
		WriteTimeout:   1 * time.Second,
		ReadTimeout:    3 * time.Second,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
		Rcodes:         true,
		Recurse:        true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := bench.Run(ctx)

	assert.Error(t, err, "expected error from benchmark run")
}

func Test_download_external_datasource_using_http_wrong_response(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	bench := Benchmark{
		Queries:        []string{ts.URL},
		Types:          []string{"A", "AAAA"},
		Server:         "8.8.8.8",
		TCP:            false,
		Concurrency:    2,
		Count:          1,
		Probability:    1,
		WriteTimeout:   1 * time.Second,
		ReadTimeout:    3 * time.Second,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
		Rcodes:         true,
		Recurse:        true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := bench.Run(ctx)

	assert.Error(t, err, "expected error from benchmark run")
}

func Test_do_classic_dns_with_duration(t *testing.T) {
	s := NewServer("udp", nil, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))
		w.WriteMsg(ret)
	})
	defer s.Close()

	bench := Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A"},
		Server:         s.Addr,
		Concurrency:    1,
		Duration:       2 * time.Second,
		Probability:    1,
		WriteTimeout:   1 * time.Second,
		ReadTimeout:    3 * time.Second,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
		Rcodes:         true,
		Recurse:        true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	assert.NoError(t, err, "expected no error from benchmark run")
	assert.GreaterOrEqual(t, rs[0].Counters.Total, int64(1), "there should be atleast one execution")
}

func Test_duration_and_count_specified_at_once(t *testing.T) {
	bench := Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A"},
		Server:         "8.8.8.8",
		TCP:            false,
		Count:          1,
		Duration:       time.Second,
		Concurrency:    1,
		Probability:    1,
		WriteTimeout:   1 * time.Second,
		ReadTimeout:    3 * time.Second,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
		Rcodes:         true,
		Recurse:        true,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := bench.Run(ctx)

	assert.Error(t, err, "expected error from benchmark run")
}

func Test_do_classic_dns_default_count(t *testing.T) {
	s := NewServer("udp", nil, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		w.WriteMsg(ret)
	})
	defer s.Close()

	bench := Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A"},
		Server:         s.Addr,
		TCP:            false,
		Concurrency:    1,
		Probability:    1,
		WriteTimeout:   1 * time.Second,
		ReadTimeout:    3 * time.Second,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
		Rcodes:         true,
		Recurse:        true,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	assert.NoError(t, err, "expected no error from benchmark run")
	assert.Len(t, rs, 1, "Run(ctx) rstats")
	assert.Equal(t, int64(1), rs[0].Counters.Total)
	assert.Equal(t, int64(1), rs[0].Counters.Success)
}

func Test_do_doq(t *testing.T) {
	server := doqServer{}
	server.start()
	defer server.stop()

	bench := createBenchmark("quic://"+server.addr, true, 1)
	bench.Insecure = true

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	assert.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
}

func Test_do_dot(t *testing.T) {
	cert, err := tls.LoadX509KeyPair("test.crt", "test.key")
	require.NoError(t, err)

	certs, err := os.ReadFile("test.crt")
	require.NoError(t, err)

	pool, err := x509.SystemCertPool()
	require.NoError(t, err)

	pool.AppendCertsFromPEM(certs)
	config := tls.Config{
		ServerName:   "localhost",
		RootCAs:      pool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	server := NewServer("tcp-tls", &config, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		w.WriteMsg(ret)
	})
	defer server.Close()

	bench := createBenchmark(server.Addr, false, 1)
	bench.Insecure = true
	bench.DOT = true

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	assert.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
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
		Queries:        []string{"example.org"},
		Types:          []string{"A", "AAAA"},
		Server:         server,
		TCP:            tcp,
		Concurrency:    2,
		Count:          1,
		Probability:    prob,
		WriteTimeout:   1 * time.Second,
		ReadTimeout:    3 * time.Second,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
		Rcodes:         true,
		Recurse:        true,
	}
}

// A returns an A record from rr. It panics on errors.
func A(rr string) *dns.A { r, _ := dns.NewRR(rr); return r.(*dns.A) }

func TestBenchmark_normalize(t *testing.T) {
	tests := []struct {
		name       string
		benchmark  Benchmark
		wantServer string
		wantErr    bool
	}{
		{
			name:       "server - IPv4",
			benchmark:  Benchmark{Server: "8.8.8.8"},
			wantServer: "8.8.8.8:53",
		},
		{
			name:       "server - IPv4 with port",
			benchmark:  Benchmark{Server: "8.8.8.8:53"},
			wantServer: "8.8.8.8:53",
		},
		{
			name:       "server - IPv6",
			benchmark:  Benchmark{Server: "fddd:dddd::"},
			wantServer: "[fddd:dddd::]:53",
		},
		{
			name:       "server - IPv6",
			benchmark:  Benchmark{Server: "fddd:dddd::"},
			wantServer: "[fddd:dddd::]:53",
		},
		{
			name:       "server - IPv6 with port",
			benchmark:  Benchmark{Server: "fddd:dddd::"},
			wantServer: "[fddd:dddd::]:53",
		},
		{
			name:       "server - HTTPS url",
			benchmark:  Benchmark{Server: "https://1.1.1.1/dns-query"},
			wantServer: "https://1.1.1.1/dns-query",
		},
		{
			name:       "server - HTTP url",
			benchmark:  Benchmark{Server: "http://127.0.0.1/dns-query"},
			wantServer: "http://127.0.0.1/dns-query",
		},
		{
			name:       "server - QUIC url",
			benchmark:  Benchmark{Server: "quic://dns.adguard-dns.com"},
			wantServer: "dns.adguard-dns.com:853",
		},
		{
			name:       "server - QUIC url with port",
			benchmark:  Benchmark{Server: "quic://localhost:853"},
			wantServer: "localhost:853",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.benchmark.normalize()

			require.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantServer, tt.benchmark.Server)
		})
	}
}

func Test_global_ratelimit(t *testing.T) {
	s := NewServer("udp", nil, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 100)

		w.WriteMsg(ret)
	})
	defer s.Close()

	bench := Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A"},
		Server:         s.Addr,
		TCP:            false,
		Concurrency:    2,
		Duration:       5 * time.Second,
		Rate:           1,
		Probability:    1,
		WriteTimeout:   1 * time.Second,
		ReadTimeout:    3 * time.Second,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
		Rcodes:         true,
		Recurse:        true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	assert.NoError(t, err, "expected no error from benchmark run")
	assert.Len(t, rs, 2)
	// assert that total queries is 5 with +-1 precision, because benchmark cancellation based on duration is not that precise
	// and one worker can start the resolution before cancelling
	assert.InDelta(t, int64(5), rs[0].Counters.Total+rs[1].Counters.Total, 1.0)
}

func Test_worker_ratelimit(t *testing.T) {
	s := NewServer("udp", nil, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 100)

		w.WriteMsg(ret)
	})
	defer s.Close()

	bench := Benchmark{
		Queries:         []string{"example.org"},
		Types:           []string{"A"},
		Server:          s.Addr,
		TCP:             false,
		Concurrency:     2,
		Duration:        5 * time.Second,
		RateLimitWorker: 1,
		Probability:     1,
		WriteTimeout:    1 * time.Second,
		ReadTimeout:     3 * time.Second,
		ConnectTimeout:  1 * time.Second,
		RequestTimeout:  5 * time.Second,
		Rcodes:          true,
		Recurse:         true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	assert.NoError(t, err, "expected no error from benchmark run")
	assert.Len(t, rs, 2)
	// assert that total queries is 10 with +-2 precision,
	// because benchmark cancellation based on duration is not that precise
	// and each worker can start the resolution before cancelling
	assert.InDelta(t, int64(10), rs[0].Counters.Total+rs[1].Counters.Total, 2.0)
}
