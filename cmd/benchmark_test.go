package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBenchmark_Run_PlainDNS(t *testing.T) {
	type args struct {
		protocol string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"DNS over UDP",
			args{
				protocol: udpNetwork,
			},
		},
		{
			"DNS over TCP",
			args{
				protocol: tcpNetwork,
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

			bench := createBenchmark(s.Addr, tt.args.protocol == tcpNetwork, 1)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			rs, err := bench.Run(ctx)

			require.NoError(t, err, "expected no error from benchmark run")
			assertResult(t, rs)
		})
	}
}

func TestBenchmark_Run_PlainDNS_dnssec(t *testing.T) {
	s := NewServer(udpNetwork, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		edns0 := ret.SetEdns0(512, false)
		edns0.AuthenticatedData = true
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		w.WriteMsg(ret)
	})
	defer s.Close()

	bench := createBenchmark(s.Addr, false, 1)
	bench.DNSSEC = true

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
	for _, r := range rs {
		assert.Equal(t, r.AuthenticatedDomains, map[string]struct{}{"example.org.": {}})
	}
}

func TestBenchmark_Run_PlainDNS_edns0(t *testing.T) {
	s := NewServer(udpNetwork, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		opt := r.IsEdns0()
		if assert.NotNil(t, opt) {
			assert.EqualValues(t, opt.UDPSize(), 1024)
		}

		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.SetEdns0(defaultEdns0BufferSize, false)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		w.WriteMsg(ret)
	})
	defer s.Close()

	bench := createBenchmark(s.Addr, false, 1)
	bench.Edns0 = 1024

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
}

func TestBenchmark_Run_PlainDNS_edns0_ednsopt(t *testing.T) {
	testOpt := uint16(65518)
	testOptData := "test"
	testHexOptData := hex.EncodeToString([]byte(testOptData))

	s := NewServer(udpNetwork, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		opt := r.IsEdns0()
		if assert.NotNil(t, opt) {
			assert.EqualValues(t, opt.UDPSize(), 1024)
			expectedOpt := false
			for _, v := range opt.Option {
				if v.Option() == testOpt {
					if localOpt, ok := v.(*dns.EDNS0_LOCAL); ok {
						assert.Equal(t, testOptData, string(localOpt.Data))
						expectedOpt = true
					}
				}
			}
			assert.True(t, expectedOpt)
		}

		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.SetEdns0(defaultEdns0BufferSize, false)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		w.WriteMsg(ret)
	})
	defer s.Close()

	bench := createBenchmark(s.Addr, false, 1)
	bench.Edns0 = 1024
	bench.EdnsOpt = strconv.Itoa(int(testOpt)) + ":" + testHexOptData

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
}

func TestBenchmark_Run_DoH_post(t *testing.T) {
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
	bench.DohMethod = postMethod

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
}

func TestBenchmark_Run_DoH_get(t *testing.T) {
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
	bench.DohMethod = getMethod

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
}

func TestBenchmark_Run_DoH_http1(t *testing.T) {
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
	bench.DohProtocol = http1Proto

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
}

func TestBenchmark_Run_DoH_http2(t *testing.T) {
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

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	ts.EnableHTTP2 = true
	ts.TLS = &config
	ts.StartTLS()
	defer ts.Close()

	bench := createBenchmark(ts.URL, true, 1)
	bench.DohProtocol = http2Proto
	bench.Insecure = true

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
}

func TestBenchmark_Run_PlainDNS_probability(t *testing.T) {
	s := NewServer(udpNetwork, nil, func(w dns.ResponseWriter, r *dns.Msg) {
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

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 2, "expected results from two workers")
	assert.Equal(t, int64(0), rs[0].Counters.Total, "Run(ctx) total counter")
	assert.Equal(t, int64(0), rs[1].Counters.Total, "Run(ctx) total counter")
}

func TestBenchmark_Run_PlainDNS_download_external_datasource_using_http(t *testing.T) {
	s := NewServer(udpNetwork, nil, func(w dns.ResponseWriter, r *dns.Msg) {
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

	bench := createBenchmark(s.Addr, false, 1)
	bench.Queries = []string{ts.URL}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
}

func TestBenchmark_Run_PlainDNS_download_external_datasource_using_http_not_available(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	// close right away to get dead port
	ts.Close()

	bench := createBenchmark("8.8.8.8", false, 1)
	bench.Queries = []string{ts.URL}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := bench.Run(ctx)

	require.Error(t, err, "expected error from benchmark run")
}

func TestBenchmark_Run_PlainDNS_download_external_datasource_using_http_wrong_response(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	bench := createBenchmark("8.8.8.8", false, 1)
	bench.Queries = []string{ts.URL}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := bench.Run(ctx)

	require.Error(t, err, "expected error from benchmark run")
}

func TestBenchmark_Run_PlainDNS_duration(t *testing.T) {
	s := NewServer(udpNetwork, nil, func(w dns.ResponseWriter, r *dns.Msg) {
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

	require.NoError(t, err, "expected no error from benchmark run")
	assert.GreaterOrEqual(t, rs[0].Counters.Total, int64(1), "there should be atleast one execution")
}

func TestBenchmark_Run_PlainDNS_duration_and_count_specified_at_once(t *testing.T) {
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

	require.Error(t, err, "expected error from benchmark run")
}

func TestBenchmark_Run_PlainDNS_default_count(t *testing.T) {
	s := NewServer(udpNetwork, nil, func(w dns.ResponseWriter, r *dns.Msg) {
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

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 1, "expected results from one worker")
	assert.Equal(t, int64(1), rs[0].Counters.Total)
	assert.Equal(t, int64(1), rs[0].Counters.Success)
}

func TestBenchmark_Run_DoQ(t *testing.T) {
	server := newDoQServer(func(r *dns.Msg) *dns.Msg {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)
		return ret
	})
	server.start()
	defer server.stop()

	bench := createBenchmark("quic://"+server.addr, true, 1)
	bench.Insecure = true

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
}

func TestBenchmark_Run_DoT(t *testing.T) {
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

	server := NewServer(tcptlsNetwork, &config, func(w dns.ResponseWriter, r *dns.Msg) {
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

	require.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
}

func TestBenchmark_prepare(t *testing.T) {
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
			name:       "server - DoT with IP address",
			benchmark:  Benchmark{Server: "8.8.8.8", DOT: true},
			wantServer: "8.8.8.8:853",
		},
		{
			name:       "server - HTTPS url",
			benchmark:  Benchmark{Server: "https://1.1.1.1"},
			wantServer: "https://1.1.1.1/dns-query",
		},
		{
			name:       "server - HTTP url",
			benchmark:  Benchmark{Server: "http://127.0.0.1"},
			wantServer: "http://127.0.0.1/dns-query",
		},
		{
			name:       "server - custom HTTP url",
			benchmark:  Benchmark{Server: "http://127.0.0.1/custom/dns-query"},
			wantServer: "http://127.0.0.1/custom/dns-query",
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
		{
			name:      "count and duration specified at once",
			benchmark: Benchmark{Server: "8.8.8.8", Count: 10, Duration: time.Minute},
			wantErr:   true,
		},
		{
			name:      "invalid EDNS0 buffer size",
			benchmark: Benchmark{Server: "8.8.8.8", Edns0: 1},
			wantErr:   true,
		},
		{
			name:      "Missing server",
			benchmark: Benchmark{},
			wantErr:   true,
		},
		{
			name:      "invalid format of ednsopt",
			benchmark: Benchmark{Server: "8.8.8.8", EdnsOpt: "test"},
			wantErr:   true,
		},
		{
			name:      "invalid format of ednsopt, code is not decimal",
			benchmark: Benchmark{Server: "8.8.8.8", EdnsOpt: "test:74657374"},
			wantErr:   true,
		},
		{
			name:      "invalid format of ednsopt, data is not hexadecimal string",
			benchmark: Benchmark{Server: "8.8.8.8", EdnsOpt: "65518:test"},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.benchmark.prepare()

			require.Equal(t, tt.wantErr, err != nil)
			if !tt.wantErr {
				assert.Equal(t, tt.wantServer, tt.benchmark.Server)
			}
		})
	}
}

func TestBenchmark_Run_global_ratelimit(t *testing.T) {
	s := NewServer(udpNetwork, nil, func(w dns.ResponseWriter, r *dns.Msg) {
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

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 2, "expected results from two workers")
	// assert that total queries is 5 with +-1 precision, because benchmark cancellation based on duration is not that precise
	// and one worker can start the resolution before cancelling
	assert.InDelta(t, int64(5), rs[0].Counters.Total+rs[1].Counters.Total, 1.0)
}

func TestBenchmark_Run_worker_ratelimit(t *testing.T) {
	s := NewServer(udpNetwork, nil, func(w dns.ResponseWriter, r *dns.Msg) {
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

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 2, "expected results from two workers")

	// assert that total queries is 10 with +-2 precision,
	// because benchmark cancellation based on duration is not that precise
	// and each worker can start the resolution before cancelling
	assert.InDelta(t, int64(10), rs[0].Counters.Total+rs[1].Counters.Total, 2.0)
}

func TestBenchmark_Run_PlainDNS_error(t *testing.T) {
	s := NewServer(udpNetwork, nil, func(w dns.ResponseWriter, r *dns.Msg) {
	})
	defer s.Close()

	bench := createBenchmark(s.Addr, false, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 2, "expected results from two workers")

	assert.Equal(t, rs[0].Counters.Total, int64(2), "there should be executions")
	assert.Equal(t, rs[0].Counters.IOError, int64(2), "there should be errors")
	assert.Equal(t, rs[1].Counters.Total, int64(2), "there should be executions")
	assert.Equal(t, rs[1].Counters.IOError, int64(2), "there should be errors")
}

func TestBenchmark_Run_DoT_error(t *testing.T) {
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

	server := NewServer(tcptlsNetwork, &config, func(w dns.ResponseWriter, r *dns.Msg) {
	})
	defer server.Close()

	bench := createBenchmark(server.Addr, false, 1)
	bench.Insecure = true
	bench.DOT = true

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 2, "expected results from two workers")

	assert.Equal(t, rs[0].Counters.Total, int64(2), "there should be executions")
	assert.Equal(t, rs[0].Counters.IOError, int64(2), "there should be errors")
	assert.Equal(t, rs[1].Counters.Total, int64(2), "there should be executions")
	assert.Equal(t, rs[1].Counters.IOError, int64(2), "there should be errors")
}

func TestBenchmark_Run_DoH_error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer ts.Close()

	bench := createBenchmark(ts.URL, true, 1)
	bench.DohMethod = postMethod

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 2, "expected results from two workers")

	assert.Equal(t, rs[0].Counters.Total, int64(2), "there should be executions")
	assert.Equal(t, rs[0].Counters.IOError, int64(2), "there should be errors")
	assert.Equal(t, rs[1].Counters.Total, int64(2), "there should be executions")
	assert.Equal(t, rs[1].Counters.IOError, int64(2), "there should be errors")
}

func TestBenchmark_Run_DoQ_error(t *testing.T) {
	server := newDoQServer(func(r *dns.Msg) *dns.Msg {
		return nil
	})
	server.start()
	defer server.stop()

	bench := createBenchmark("quic://"+server.addr, true, 1)
	bench.Insecure = true

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 2, "expected results from two workers")

	assert.Equal(t, rs[0].Counters.Total, int64(2), "there should be executions")
	assert.Equal(t, rs[0].Counters.IOError, int64(2), "there should be errors")
	assert.Equal(t, rs[1].Counters.Total, int64(2), "there should be executions")
	assert.Equal(t, rs[1].Counters.IOError, int64(2), "there should be errors")
}

func TestBenchmark_Run_PlainDNS_truncated(t *testing.T) {
	s := NewServer(udpNetwork, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))
		ret.Truncated = true

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		w.WriteMsg(ret)
	})
	defer s.Close()

	bench := createBenchmark(s.Addr, false, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 2, "expected results from two workers")

	assert.Equal(t, rs[0].Counters.Total, int64(2), "there should be executions")
	assert.Equal(t, rs[0].Counters.Truncated, int64(2), "there should be truncated messages")
	assert.Equal(t, rs[1].Counters.Total, int64(2), "there should be executions")
	assert.Equal(t, rs[1].Counters.Truncated, int64(2), "there should be truncated messages")
}

func TestBenchmark_Run_DoT_truncated(t *testing.T) {
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

	server := NewServer(tcptlsNetwork, &config, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))
		ret.Truncated = true

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

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 2, "expected results from two workers")

	assert.Equal(t, rs[0].Counters.Total, int64(2), "there should be executions")
	assert.Equal(t, rs[0].Counters.Truncated, int64(2), "there should be truncated messages")
	assert.Equal(t, rs[1].Counters.Total, int64(2), "there should be executions")
	assert.Equal(t, rs[1].Counters.Truncated, int64(2), "there should be truncated messages")
}

func TestBenchmark_Run_DoH_truncated(t *testing.T) {
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

		ret := new(dns.Msg)
		ret.SetReply(&msg)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))
		ret.Truncated = true

		pack, err := ret.Pack()
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
	bench.DohMethod = postMethod

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 2, "expected results from two workers")

	assert.Equal(t, rs[0].Counters.Total, int64(2), "there should be executions")
	assert.Equal(t, rs[0].Counters.Truncated, int64(2), "there should be truncated messages")
	assert.Equal(t, rs[1].Counters.Total, int64(2), "there should be executions")
	assert.Equal(t, rs[1].Counters.Truncated, int64(2), "there should be truncated messages")
}

func TestBenchmark_Run_DoQ_truncated(t *testing.T) {
	server := newDoQServer(func(r *dns.Msg) *dns.Msg {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))
		ret.Truncated = true

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)
		return ret
	})
	server.start()
	defer server.stop()

	bench := createBenchmark("quic://"+server.addr, true, 1)
	bench.Insecure = true

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 2, "expected results from two workers")

	assert.Equal(t, rs[0].Counters.Total, int64(2), "there should be executions")
	assert.Equal(t, rs[0].Counters.Truncated, int64(2), "there should be truncated messages")
	assert.Equal(t, rs[1].Counters.Total, int64(2), "there should be executions")
	assert.Equal(t, rs[1].Counters.Truncated, int64(2), "there should be truncated messages")
}

func ExampleBenchmark_Run_plainDNS_udp() {
	bench := createBenchmark("8.8.8.8", false, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	bench.Run(ctx)

	// Output: Using 1 hostnames
	// Benchmarking 8.8.8.8:53 via udp with 2 concurrent requests
}

func ExampleBenchmark_Run_plainDNS_tcp() {
	bench := createBenchmark("8.8.8.8", true, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	bench.Run(ctx)

	// Output: Using 1 hostnames
	// Benchmarking 8.8.8.8:53 via tcp with 2 concurrent requests
}

func ExampleBenchmark_Run_dot() {
	bench := createBenchmark("8.8.8.8", true, 1)
	bench.DOT = true

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	bench.Run(ctx)

	// Output: Using 1 hostnames
	// Benchmarking 8.8.8.8:853 via tcp-tls with 2 concurrent requests
}

func ExampleBenchmark_Run_doh() {
	bench := createBenchmark("https://1.1.1.1", true, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	bench.Run(ctx)

	// Output: Using 1 hostnames
	// Benchmarking https://1.1.1.1/dns-query via https/1.1 (POST) with 2 concurrent requests
}

func ExampleBenchmark_Run_doh_get() {
	bench := createBenchmark("https://1.1.1.1", true, 1)
	bench.DohMethod = getMethod

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	bench.Run(ctx)

	// Output: Using 1 hostnames
	// Benchmarking https://1.1.1.1/dns-query via https/1.1 (GET) with 2 concurrent requests
}

func ExampleBenchmark_Run_doh_http2() {
	bench := createBenchmark("https://1.1.1.1", true, 1)
	bench.DohProtocol = http2Proto

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	bench.Run(ctx)

	// Output: Using 1 hostnames
	// Benchmarking https://1.1.1.1/dns-query via https/2 (POST) with 2 concurrent requests
}

func ExampleBenchmark_Run_doh_http3() {
	bench := createBenchmark("https://1.1.1.1", true, 1)
	bench.DohProtocol = http3Proto

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	bench.Run(ctx)

	// Output: Using 1 hostnames
	// Benchmarking https://1.1.1.1/dns-query via https/3 (POST) with 2 concurrent requests
}

func ExampleBenchmark_Run_doq() {
	bench := createBenchmark("quic://dns.adguard-dns.com", true, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	bench.Run(ctx)

	// Output: Using 1 hostnames
	// Benchmarking dns.adguard-dns.com:853 via quic with 2 concurrent requests
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
