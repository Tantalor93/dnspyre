package dnsbench_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/suite"
	"github.com/tantalor93/dnspyre/v3/pkg/dnsbench"
)

type DoHTestSuite struct {
	suite.Suite
}

func TestDoHSuite(t *testing.T) {
	suite.Run(t, new(DoHTestSuite))
}

func (suite *DoHTestSuite) TestBenchmark_Run_post() {
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
		Queries:     []string{"example.org"},
		Types:       []string{"A", "AAAA"},
		Server:      ts.URL,
		TCP:         true,
		Concurrency: 2,
		Rcodes:      true,
		Recurse:     true,
		DohMethod:   dnsbench.PostHTTPMethod,
		Writer:      &buf,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	suite.Require().NoError(err, "expected no error from benchmark run")
	assertResult(suite.T(), rs)
	suite.Equal(fmt.Sprintf("Using 1 hostnames\nBenchmarking %s/dns-query via http/1.1 (POST) with 2 concurrent requests \n", ts.URL), buf.String())
}

func (suite *DoHTestSuite) TestBenchmark_Run_get() {
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
		Queries:     []string{"example.org"},
		Types:       []string{"A", "AAAA"},
		Server:      ts.URL,
		TCP:         true,
		Concurrency: 2,
		Rcodes:      true,
		Recurse:     true,
		DohMethod:   dnsbench.GetHTTPMethod,
		Writer:      &buf,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	suite.Require().NoError(err, "expected no error from benchmark run")
	assertResult(suite.T(), rs)
	suite.Equal(fmt.Sprintf("Using 1 hostnames\nBenchmarking %s/dns-query via http/1.1 (GET) with 2 concurrent requests \n", ts.URL), buf.String())
}

func (suite *DoHTestSuite) TestBenchmark_Run_http1() {
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
		Queries:     []string{"example.org"},
		Types:       []string{"A", "AAAA"},
		Server:      ts.URL,
		TCP:         true,
		Concurrency: 2,
		Rcodes:      true,
		Recurse:     true,
		DohProtocol: dnsbench.HTTP1Proto,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	suite.Require().NoError(err, "expected no error from benchmark run")
	assertResult(suite.T(), rs)
}

func (suite *DoHTestSuite) TestBenchmark_Run_http2() {
	cert, err := tls.LoadX509KeyPair("testdata/test.crt", "testdata/test.key")
	suite.Require().NoError(err)

	certs, err := os.ReadFile("testdata/test.crt")
	suite.Require().NoError(err)

	pool, err := x509.SystemCertPool()
	suite.Require().NoError(err)

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
		Queries:     []string{"example.org"},
		Types:       []string{"A", "AAAA"},
		Server:      ts.URL,
		TCP:         true,
		Concurrency: 2,
		Rcodes:      true,
		Recurse:     true,
		DohProtocol: dnsbench.HTTP2Proto,
		Insecure:    true,
		Writer:      &buf,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	suite.Require().NoError(err, "expected no error from benchmark run")
	assertResult(suite.T(), rs)
	suite.Equal(fmt.Sprintf("Using 1 hostnames\nBenchmarking %s/dns-query via https/2 (POST) with 2 concurrent requests \n", ts.URL), buf.String())
}

func (suite *DoHTestSuite) TestBenchmark_Run_truncated() {
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
		Queries:     []string{"example.org"},
		Types:       []string{"A", "AAAA"},
		Server:      ts.URL,
		TCP:         true,
		Concurrency: 2,
		Rcodes:      true,
		Recurse:     true,
		DohMethod:   dnsbench.PostHTTPMethod,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	suite.Require().NoError(err, "expected no error from benchmark run")
	suite.Require().Len(rs, 2, "expected results from two workers")

	suite.EqualValues(2, rs[0].Counters.Total, "there should be executions")
	suite.EqualValues(2, rs[0].Counters.Truncated, "there should be truncated messages")
	suite.EqualValues(2, rs[1].Counters.Total, "there should be executions")
	suite.EqualValues(2, rs[1].Counters.Truncated, "there should be truncated messages")
}

func (suite *DoHTestSuite) TestBenchmark_Run_error() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	bench := dnsbench.Benchmark{
		Queries:     []string{"example.org"},
		Types:       []string{"A", "AAAA"},
		Server:      ts.URL,
		TCP:         true,
		Concurrency: 2,
		Rcodes:      true,
		Recurse:     true,
		DohMethod:   dnsbench.PostHTTPMethod,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	suite.Require().NoError(err, "expected no error from benchmark run")
	suite.Require().Len(rs, 2, "expected results from two workers")

	suite.EqualValues(2, rs[0].Counters.Total, "there should be executions")
	suite.EqualValues(2, rs[0].Counters.IOError, "there should be errors")
	suite.EqualValues(2, rs[1].Counters.Total, "there should be executions")
	suite.EqualValues(2, rs[1].Counters.IOError, "there should be errors")
}

func (suite *DoHTestSuite) TestBenchmark_Run_separate_connections() {
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
		suite.Run(tt.name, func() {
			cert, err := tls.LoadX509KeyPair("testdata/test.crt", "testdata/test.key")
			suite.Require().NoError(err)

			certs, err := os.ReadFile("testdata/test.crt")
			suite.Require().NoError(err)

			pool, err := x509.SystemCertPool()
			suite.Require().NoError(err)

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
				Server:                    ts.URL,
				DohProtocol:               "2",
				TCP:                       true,
				Concurrency:               5,
				Count:                     2,
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

			suite.Require().NoError(err, "expected no error from benchmark run")
			suite.Require().Len(rs, 5)
			for _, v := range rs {
				suite.Empty(v.Errors)
			}
			suite.Len(remoteAddrs, tt.wantNumberOfConnections)
			suite.Equal(fmt.Sprintf("Using 1 hostnames\nBenchmarking %s/dns-query via https/2 (POST) with 5 concurrent requests \n", ts.URL), buf.String())
		})
	}
}
