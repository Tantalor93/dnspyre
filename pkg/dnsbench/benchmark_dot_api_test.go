package dnsbench_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/suite"
	"github.com/tantalor93/dnspyre/v3/pkg/dnsbench"
)

type DoTTestSuite struct {
	suite.Suite
}

func TestDoTTestSuite(t *testing.T) {
	suite.Run(t, new(DoTTestSuite))
}

func (suite *DoTTestSuite) TestBenchmark_Run() {
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
		Queries:     []string{"example.org"},
		Types:       []string{"A", "AAAA"},
		Server:      server.Addr,
		TCP:         false,
		Concurrency: 2,
		Rcodes:      true,
		Recurse:     true,
		Insecure:    true,
		DOT:         true,
		Writer:      &buf,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	suite.Require().NoError(err, "expected no error from benchmark run")
	assertResult(suite.T(), rs)
	suite.Equal(fmt.Sprintf("Using 1 hostnames\nBenchmarking %s via tcp-tls with 2 concurrent requests \n", server.Addr), buf.String())
}

func (suite *DoTTestSuite) TestBenchmark_Run_truncated() {
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
		Queries:     []string{"example.org"},
		Types:       []string{"A", "AAAA"},
		Server:      server.Addr,
		TCP:         false,
		Concurrency: 2,
		Rcodes:      true,
		Recurse:     true,
		Insecure:    true,
		DOT:         true,
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

func (suite *DoTTestSuite) TestBenchmark_Run_error() {
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

	server := NewServer(dnsbench.TLSTransport, &config, func(_ dns.ResponseWriter, _ *dns.Msg) {
	})
	defer server.Close()

	bench := dnsbench.Benchmark{
		Queries:        []string{"example.org"},
		Types:          []string{"A", "AAAA"},
		Server:         server.Addr,
		TCP:            false,
		Concurrency:    2,
		WriteTimeout:   100 * time.Millisecond,
		ReadTimeout:    300 * time.Millisecond,
		ConnectTimeout: 100 * time.Millisecond,
		RequestTimeout: 500 * time.Millisecond,
		Rcodes:         true,
		Recurse:        true,
		Insecure:       true,
		DOT:            true,
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
