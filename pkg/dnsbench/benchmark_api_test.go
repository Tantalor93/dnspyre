package dnsbench_test

import (
	"bufio"
	"bytes"
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
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/quic-go/quic-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tantalor93/dnspyre/v3/pkg/dnsbench"
)

func TestBenchmark_Run_PlainDNS(t *testing.T) {
	type args struct {
		protocol string
	}
	tests := []struct {
		name               string
		args               args
		wantOutputTemplate string
	}{
		{
			name: "DNS over UDP",
			args: args{
				protocol: dnsbench.UDPTransport,
			},
			wantOutputTemplate: "Using 1 hostnames\nBenchmarking %s via udp with 2 concurrent requests \n",
		},
		{
			name: "DNS over TCP",
			args: args{
				protocol: dnsbench.TCPTransport,
			},
			wantOutputTemplate: "Using 1 hostnames\nBenchmarking %s via tcp with 2 concurrent requests \n",
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

			buf := bytes.Buffer{}
			bench := dnsbench.Benchmark{
				Queries:        []string{"example.org"},
				Types:          []string{"A", "AAAA"},
				Server:         s.Addr,
				TCP:            tt.args.protocol == dnsbench.TCPTransport,
				Concurrency:    2,
				Count:          1,
				Probability:    1,
				WriteTimeout:   1 * time.Second,
				ReadTimeout:    3 * time.Second,
				ConnectTimeout: 1 * time.Second,
				RequestTimeout: 5 * time.Second,
				Rcodes:         true,
				Recurse:        true,
				Writer:         &buf,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			rs, err := bench.Run(ctx)

			require.NoError(t, err, "expected no error from benchmark run")
			assertResult(t, rs)
			assert.Equal(t, fmt.Sprintf(tt.wantOutputTemplate, s.Addr), buf.String())
		})
	}
}

func TestBenchmark_Run_PlainDNS_dnssec(t *testing.T) {
	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
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

	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
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
		DNSSEC:         true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
	for _, r := range rs {
		assert.Equal(t, map[string]struct{}{"example.org.": {}}, r.AuthenticatedDomains)
	}
}

func TestBenchmark_Run_PlainDNS_edns0(t *testing.T) {
	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		opt := r.IsEdns0()
		if assert.NotNil(t, opt) {
			assert.EqualValues(t, 1024, opt.UDPSize())
		}

		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.SetEdns0(dnsbench.DefaultEdns0BufferSize, false)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		w.WriteMsg(ret)
	})
	defer s.Close()

	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
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
		Edns0:          1024,
	}

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

	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		opt := r.IsEdns0()
		if assert.NotNil(t, opt) {
			assert.EqualValues(t, 1024, opt.UDPSize())
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
		ret.SetEdns0(dnsbench.DefaultEdns0BufferSize, false)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		w.WriteMsg(ret)
	})
	defer s.Close()

	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
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
		Edns0:          1024,
		EdnsOpt:        strconv.Itoa(int(testOpt)) + ":" + testHexOptData,
	}

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

	buf := bytes.Buffer{}
	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A", "AAAA"},
		Server:         ts.URL,
		TCP:            true,
		Concurrency:    2,
		Count:          1,
		Probability:    1,
		WriteTimeout:   1 * time.Second,
		ReadTimeout:    3 * time.Second,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
		Rcodes:         true,
		Recurse:        true,
		DohMethod:      dnsbench.PostHTTPMethod,
		Writer:         &buf,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
	assert.Equal(t, fmt.Sprintf("Using 1 hostnames\nBenchmarking %s/dns-query via http/1.1 (POST) with 2 concurrent requests \n", ts.URL), buf.String())
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

	buf := bytes.Buffer{}
	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A", "AAAA"},
		Server:         ts.URL,
		TCP:            true,
		Concurrency:    2,
		Count:          1,
		Probability:    1,
		WriteTimeout:   1 * time.Second,
		ReadTimeout:    3 * time.Second,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
		Rcodes:         true,
		Recurse:        true,
		DohMethod:      dnsbench.GetHTTPMethod,
		Writer:         &buf,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
	assert.Equal(t, fmt.Sprintf("Using 1 hostnames\nBenchmarking %s/dns-query via http/1.1 (GET) with 2 concurrent requests \n", ts.URL), buf.String())
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

	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A", "AAAA"},
		Server:         ts.URL,
		TCP:            true,
		Concurrency:    2,
		Count:          1,
		Probability:    1,
		WriteTimeout:   1 * time.Second,
		ReadTimeout:    3 * time.Second,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
		Rcodes:         true,
		Recurse:        true,
		DohProtocol:    dnsbench.HTTP1Proto,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
}

func TestBenchmark_Run_DoH_http2(t *testing.T) {
	cert, err := tls.LoadX509KeyPair("testdata/test.crt", "testdata/test.key")
	require.NoError(t, err)

	certs, err := os.ReadFile("testdata/test.crt")
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

	buf := bytes.Buffer{}
	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A", "AAAA"},
		Server:         ts.URL,
		TCP:            true,
		Concurrency:    2,
		Count:          1,
		Probability:    1,
		WriteTimeout:   1 * time.Second,
		ReadTimeout:    3 * time.Second,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
		Rcodes:         true,
		Recurse:        true,
		DohProtocol:    dnsbench.HTTP2Proto,
		Insecure:       true,
		Writer:         &buf,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
	assert.Equal(t, fmt.Sprintf("Using 1 hostnames\nBenchmarking %s/dns-query via https/2 (POST) with 2 concurrent requests \n", ts.URL), buf.String())
}

func TestBenchmark_Run_PlainDNS_probability(t *testing.T) {
	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		w.WriteMsg(ret)
	})
	defer s.Close()

	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A", "AAAA"},
		Server:         s.Addr,
		TCP:            false,
		Concurrency:    2,
		Count:          1,
		Probability:    0,
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
	assert.Equal(t, int64(0), rs[0].Counters.Total, "Run(ctx) total counter")
	assert.Equal(t, int64(0), rs[1].Counters.Total, "Run(ctx) total counter")
}

func TestBenchmark_Run_PlainDNS_download_external_datasource_using_http(t *testing.T) {
	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		w.WriteMsg(ret)
	})
	defer s.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`example.org`))
		if err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	bench := dnsbench.Benchmark{
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

	require.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
}

func TestBenchmark_Run_PlainDNS_download_external_datasource_using_http_not_available(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
	}))
	// close right away to get dead port
	ts.Close()

	bench := dnsbench.Benchmark{
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

	require.Error(t, err, "expected error from benchmark run")
}

func TestBenchmark_Run_PlainDNS_download_external_datasource_using_http_wrong_response(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	bench := dnsbench.Benchmark{
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

	require.Error(t, err, "expected error from benchmark run")
}

func TestBenchmark_Run_PlainDNS_duration(t *testing.T) {
	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))
		w.WriteMsg(ret)
	})
	defer s.Close()

	bench := dnsbench.Benchmark{
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
	bench := dnsbench.Benchmark{
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
	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		w.WriteMsg(ret)
	})
	defer s.Close()

	bench := dnsbench.Benchmark{
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
	server := newDoQServer(func(_ quic.Connection, r *dns.Msg) *dns.Msg {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)
		return ret
	})
	server.start()
	defer server.stop()

	buf := bytes.Buffer{}
	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A", "AAAA"},
		Server:         "quic://" + server.addr,
		TCP:            true,
		Concurrency:    2,
		Count:          1,
		Probability:    1,
		WriteTimeout:   1 * time.Second,
		ReadTimeout:    3 * time.Second,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
		Rcodes:         true,
		Recurse:        true,
		Insecure:       true,
		Writer:         &buf,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
	assert.Equal(t, fmt.Sprintf("Using 1 hostnames\nBenchmarking %s via quic with 2 concurrent requests \n", server.addr), buf.String())
}

func TestBenchmark_Run_DoT(t *testing.T) {
	cert, err := tls.LoadX509KeyPair("testdata/test.crt", "testdata/test.key")
	require.NoError(t, err)

	certs, err := os.ReadFile("testdata/test.crt")
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

	server := NewServer(dnsbench.TLSTransport, &config, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		w.WriteMsg(ret)
	})
	defer server.Close()

	buf := bytes.Buffer{}
	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A", "AAAA"},
		Server:         server.Addr,
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
		Insecure:       true,
		DOT:            true,
		Writer:         &buf,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)
	assert.Equal(t, fmt.Sprintf("Using 1 hostnames\nBenchmarking %s via tcp-tls with 2 concurrent requests \n", server.Addr), buf.String())
}

func TestBenchmark_Run_global_ratelimit(t *testing.T) {
	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 100)

		w.WriteMsg(ret)
	})
	defer s.Close()

	bench := dnsbench.Benchmark{
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
	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 100)

		w.WriteMsg(ret)
	})
	defer s.Close()

	bench := dnsbench.Benchmark{
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
	s := NewServer(dnsbench.UDPTransport, nil, func(_ dns.ResponseWriter, _ *dns.Msg) {
	})
	defer s.Close()

	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
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

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 2, "expected results from two workers")

	assert.Equal(t, int64(2), rs[0].Counters.Total, "there should be executions")
	assert.Equal(t, int64(2), rs[0].Counters.IOError, "there should be errors")
	assert.Equal(t, int64(2), rs[1].Counters.Total, "there should be executions")
	assert.Equal(t, int64(2), rs[1].Counters.IOError, "there should be errors")
}

func TestBenchmark_Run_DoT_error(t *testing.T) {
	cert, err := tls.LoadX509KeyPair("testdata/test.crt", "testdata/test.key")
	require.NoError(t, err)

	certs, err := os.ReadFile("testdata/test.crt")
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

	server := NewServer(dnsbench.TLSTransport, &config, func(_ dns.ResponseWriter, _ *dns.Msg) {
	})
	defer server.Close()

	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A", "AAAA"},
		Server:         server.Addr,
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
		Insecure:       true,
		DOT:            true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 2, "expected results from two workers")

	assert.Equal(t, int64(2), rs[0].Counters.Total, "there should be executions")
	assert.Equal(t, int64(2), rs[0].Counters.IOError, "there should be errors")
	assert.Equal(t, int64(2), rs[1].Counters.Total, "there should be executions")
	assert.Equal(t, int64(2), rs[1].Counters.IOError, "there should be errors")
}

func TestBenchmark_Run_DoH_error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A", "AAAA"},
		Server:         ts.URL,
		TCP:            true,
		Concurrency:    2,
		Count:          1,
		Probability:    1,
		WriteTimeout:   1 * time.Second,
		ReadTimeout:    3 * time.Second,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
		Rcodes:         true,
		Recurse:        true,
		DohMethod:      dnsbench.PostHTTPMethod,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 2, "expected results from two workers")

	assert.Equal(t, int64(2), rs[0].Counters.Total, "there should be executions")
	assert.Equal(t, int64(2), rs[0].Counters.IOError, "there should be errors")
	assert.Equal(t, int64(2), rs[1].Counters.Total, "there should be executions")
	assert.Equal(t, int64(2), rs[1].Counters.IOError, "there should be errors")
}

func TestBenchmark_Run_DoQ_error(t *testing.T) {
	server := newDoQServer(func(_ quic.Connection, _ *dns.Msg) *dns.Msg {
		return nil
	})
	server.start()
	defer server.stop()

	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A", "AAAA"},
		Server:         "quic://" + server.addr,
		TCP:            true,
		Concurrency:    2,
		Count:          1,
		Probability:    1,
		WriteTimeout:   1 * time.Second,
		ReadTimeout:    3 * time.Second,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
		Rcodes:         true,
		Recurse:        true,
		Insecure:       true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 2, "expected results from two workers")

	assert.Equal(t, int64(2), rs[0].Counters.Total, "there should be executions")
	assert.Equal(t, int64(2), rs[0].Counters.IOError, "there should be errors")
	assert.Equal(t, int64(2), rs[1].Counters.Total, "there should be executions")
	assert.Equal(t, int64(2), rs[1].Counters.IOError, "there should be errors")
}

func TestBenchmark_Run_PlainDNS_truncated(t *testing.T) {
	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))
		ret.Truncated = true

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		w.WriteMsg(ret)
	})
	defer s.Close()

	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
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

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 2, "expected results from two workers")

	assert.Equal(t, int64(2), rs[0].Counters.Total, "there should be executions")
	assert.Equal(t, int64(2), rs[0].Counters.Truncated, "there should be truncated messages")
	assert.Equal(t, int64(2), rs[1].Counters.Total, "there should be executions")
	assert.Equal(t, int64(2), rs[1].Counters.Truncated, "there should be truncated messages")
}

func TestBenchmark_Run_DoT_truncated(t *testing.T) {
	cert, err := tls.LoadX509KeyPair("testdata/test.crt", "testdata/test.key")
	require.NoError(t, err)

	certs, err := os.ReadFile("testdata/test.crt")
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

	server := NewServer(dnsbench.TLSTransport, &config, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))
		ret.Truncated = true

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		w.WriteMsg(ret)
	})
	defer server.Close()

	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A", "AAAA"},
		Server:         server.Addr,
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
		Insecure:       true,
		DOT:            true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 2, "expected results from two workers")

	assert.Equal(t, int64(2), rs[0].Counters.Total, "there should be executions")
	assert.Equal(t, int64(2), rs[0].Counters.Truncated, "there should be truncated messages")
	assert.Equal(t, int64(2), rs[1].Counters.Total, "there should be executions")
	assert.Equal(t, int64(2), rs[1].Counters.Truncated, "there should be truncated messages")
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

	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A", "AAAA"},
		Server:         ts.URL,
		TCP:            true,
		Concurrency:    2,
		Count:          1,
		Probability:    1,
		WriteTimeout:   1 * time.Second,
		ReadTimeout:    3 * time.Second,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
		Rcodes:         true,
		Recurse:        true,
		DohMethod:      dnsbench.PostHTTPMethod,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 2, "expected results from two workers")

	assert.Equal(t, int64(2), rs[0].Counters.Total, "there should be executions")
	assert.Equal(t, int64(2), rs[0].Counters.Truncated, "there should be truncated messages")
	assert.Equal(t, int64(2), rs[1].Counters.Total, "there should be executions")
	assert.Equal(t, int64(2), rs[1].Counters.Truncated, "there should be truncated messages")
}

func TestBenchmark_Run_DoQ_truncated(t *testing.T) {
	server := newDoQServer(func(_ quic.Connection, r *dns.Msg) *dns.Msg {
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

	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A", "AAAA"},
		Server:         "quic://" + server.addr,
		TCP:            true,
		Concurrency:    2,
		Count:          1,
		Probability:    1,
		WriteTimeout:   1 * time.Second,
		ReadTimeout:    3 * time.Second,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
		Rcodes:         true,
		Recurse:        true,
		Insecure:       true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	require.Len(t, rs, 2, "expected results from two workers")

	assert.Equal(t, int64(2), rs[0].Counters.Total, "there should be executions")
	assert.Equal(t, int64(2), rs[0].Counters.Truncated, "there should be truncated messages")
	assert.Equal(t, int64(2), rs[1].Counters.Total, "there should be executions")
	assert.Equal(t, int64(2), rs[1].Counters.Truncated, "there should be truncated messages")
}

type requestLog struct {
	worker    int
	requestid int
	qname     string
	qtype     string
	respid    int
	rcode     string
	respflags string
	err       string
	duration  time.Duration
}

func TestBenchmark_Requestlog(t *testing.T) {
	requestLogPath := t.TempDir() + "/requests.log"

	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 500)

		w.WriteMsg(ret)
	})
	defer s.Close()

	buf := bytes.Buffer{}
	bench := dnsbench.Benchmark{
		Queries:           []string{"example.org"},
		Types:             []string{"A", "AAAA"},
		Server:            s.Addr,
		TCP:               false,
		Concurrency:       2,
		Count:             1,
		Probability:       1,
		WriteTimeout:      1 * time.Second,
		ReadTimeout:       3 * time.Second,
		ConnectTimeout:    1 * time.Second,
		RequestTimeout:    5 * time.Second,
		Rcodes:            true,
		Recurse:           true,
		Writer:            &buf,
		RequestLogEnabled: true,
		RequestLogPath:    requestLogPath,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	require.NoError(t, err, "expected no error from benchmark run")
	assertResult(t, rs)

	requestLogFile, err := os.Open(requestLogPath)
	require.NoError(t, err)

	requestLogs := parseRequestLogs(t, requestLogFile)

	workerIDs := map[int]int{}
	qtypes := map[string]int{}

	for _, v := range requestLogs {
		workerIDs[v.worker]++
		qtypes[v.qtype]++

		assert.Equal(t, "example.org.", v.qname)
		assert.NotZero(t, v.requestid)
		assert.NotZero(t, v.respid)
		assert.Equal(t, "NOERROR", v.rcode)
		assert.Equal(t, "qr rd", v.respflags)
		assert.Equal(t, "<nil>", v.err)
		assert.NotZero(t, v.duration)
	}
	assert.Equal(t, map[int]int{0: 2, 1: 2}, workerIDs)
	assert.Equal(t, map[string]int{"AAAA": 2, "A": 2}, qtypes)
}

func TestBenchmark_Run_DoH_separate_connections(t *testing.T) {
	tests := []struct {
		name                    string
		separateConnections     bool
		wantNumberOfConnections int
	}{
		{
			name:                    "separate connections",
			separateConnections:     true,
			wantNumberOfConnections: 5,
		},
		{
			name:                    "shared connections",
			separateConnections:     false,
			wantNumberOfConnections: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert, err := tls.LoadX509KeyPair("testdata/test.crt", "testdata/test.key")
			require.NoError(t, err)

			certs, err := os.ReadFile("testdata/test.crt")
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

			mutex := sync.Mutex{}
			remoteAddrs := make(map[string]int)

			ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mutex.Lock()
				remoteAddrs[r.RemoteAddr]++
				mutex.Unlock()

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

				_, err = w.Write(pack)
				if err != nil {
					panic(err)
				}
			}))
			ts.EnableHTTP2 = true
			ts.TLS = &config
			ts.StartTLS()
			defer ts.Close()

			buf := bytes.Buffer{}
			bench := dnsbench.Benchmark{
				Queries:                   []string{"example.org"},
				Types:                     []string{"A"},
				Server:                    ts.URL,
				DohProtocol:               "2",
				TCP:                       true,
				Concurrency:               5,
				Count:                     2,
				Probability:               1,
				WriteTimeout:              1 * time.Second,
				ReadTimeout:               3 * time.Second,
				ConnectTimeout:            1 * time.Second,
				RequestTimeout:            5 * time.Second,
				Rcodes:                    true,
				Recurse:                   true,
				DohMethod:                 dnsbench.PostHTTPMethod,
				Writer:                    &buf,
				SeparateWorkerConnections: tt.separateConnections,
				Insecure:                  true,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			rs, err := bench.Run(ctx)

			// close right away to mitigate race detector failures
			ts.Close()

			require.NoError(t, err, "expected no error from benchmark run")
			assert.Len(t, rs, 5)
			for _, v := range rs {
				assert.Empty(t, v.Errors)
			}
			assert.Len(t, remoteAddrs, tt.wantNumberOfConnections)
			assert.Equal(t, fmt.Sprintf("Using 1 hostnames\nBenchmarking %s/dns-query via https/2 (POST) with 5 concurrent requests \n", ts.URL), buf.String())
		})
	}
}

func TestBenchmark_Run_DoQ_separate_connections(t *testing.T) {
	tests := []struct {
		name                    string
		separateConnections     bool
		wantNumberOfConnections int
	}{
		{
			name:                    "separate connections",
			separateConnections:     true,
			wantNumberOfConnections: 5,
		},
		{
			name:                    "shared connections",
			separateConnections:     false,
			wantNumberOfConnections: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutex := sync.Mutex{}
			remoteAddrs := make(map[string]int)

			server := newDoQServer(func(c quic.Connection, r *dns.Msg) *dns.Msg {
				mutex.Lock()
				remoteAddrs[c.RemoteAddr().String()]++
				mutex.Unlock()

				ret := new(dns.Msg)
				ret.SetReply(r)
				ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))
				return ret
			})
			server.start()
			defer server.stop()

			buf := bytes.Buffer{}
			bench := dnsbench.Benchmark{
				Queries:                   []string{"example.org"},
				Types:                     []string{"A"},
				Server:                    "quic://" + server.addr,
				TCP:                       true,
				Concurrency:               5,
				Count:                     2,
				Probability:               1,
				WriteTimeout:              1 * time.Second,
				ReadTimeout:               3 * time.Second,
				ConnectTimeout:            1 * time.Second,
				RequestTimeout:            5 * time.Second,
				Rcodes:                    true,
				Recurse:                   true,
				DohMethod:                 dnsbench.PostHTTPMethod,
				Writer:                    &buf,
				SeparateWorkerConnections: tt.separateConnections,
				Insecure:                  true,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			rs, err := bench.Run(ctx)

			// stop right away to mitigate race detector failures
			server.stop()

			require.NoError(t, err, "expected no error from benchmark run")
			assert.Len(t, rs, 5)
			for _, v := range rs {
				assert.Empty(t, v.Errors)
			}
			assert.Len(t, remoteAddrs, tt.wantNumberOfConnections)
			assert.Equal(t, fmt.Sprintf("Using 1 hostnames\nBenchmarking %s via quic with 5 concurrent requests \n", server.addr), buf.String())
		})
	}
}

func parseRequestLogs(t *testing.T, reader io.Reader) []requestLog {
	pattern := `.*worker:\[(.*)\] reqid:\[(.*)\] qname:\[(.*)\] qtype:\[(.*)\] respid:\[(.*)\] rcode:\[(.*)\] respflags:\[(.*)\] err:\[(.*)\] duration:\[(.*)\]$`
	regex := regexp.MustCompile(pattern)
	scanner := bufio.NewScanner(reader)
	var requestLogs []requestLog
	for scanner.Scan() {
		line := scanner.Text()

		matches := regex.FindStringSubmatch(line)

		workerID, err := strconv.Atoi(matches[1])
		require.NoError(t, err)

		requestID, err := strconv.Atoi(matches[2])
		require.NoError(t, err)

		qname := matches[3]
		qtype := matches[4]

		respID, err := strconv.Atoi(matches[5])
		require.NoError(t, err)

		rcode := matches[6]
		respflags := matches[7]
		errstr := matches[8]

		dur, err := time.ParseDuration(matches[9])
		require.NoError(t, err)

		requestLogs = append(requestLogs, requestLog{
			worker:    workerID,
			requestid: requestID,
			qname:     qname,
			qtype:     qtype,
			respid:    respID,
			rcode:     rcode,
			respflags: respflags,
			err:       errstr,
			duration:  dur,
		})
	}
	return requestLogs
}

func assertResult(t *testing.T, rs []*dnsbench.ResultStats) {
	if assert.Len(t, rs, 2, "Run(ctx) rstats") {
		rs0 := rs[0]
		rs1 := rs[1]
		assertResultStats(t, rs0)
		assertResultStats(t, rs1)
		assertTimings(t, rs0)
		assertTimings(t, rs1)
	}
}

func assertResultStats(t *testing.T, rs *dnsbench.ResultStats) {
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

func assertTimings(t *testing.T, rs *dnsbench.ResultStats) {
	if assert.Len(t, rs.Timings, 2, "Run(ctx) rstats timings") {
		t0 := rs.Timings[0]
		t1 := rs.Timings[1]
		assert.NotZero(t, t0.Duration, "Run(ctx) rstats timings duration")
		assert.NotZero(t, t0.Start, "Run(ctx) rstats timings start")
		assert.NotZero(t, t1.Duration, "Run(ctx) rstats timings duration")
		assert.NotZero(t, t1.Start, "Run(ctx) rstats timings start")
	}
}

// A returns an A record from rr. It panics on errors.
func A(rr string) *dns.A { r, _ := dns.NewRR(rr); return r.(*dns.A) }
