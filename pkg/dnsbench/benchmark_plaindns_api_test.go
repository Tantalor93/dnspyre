package dnsbench_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tantalor93/dnspyre/v3/pkg/dnsbench"
)

type PlainDNSTestSuite struct {
	suite.Suite
}

func TestPlainDNSTestSuite(t *testing.T) {
	suite.Run(t, new(PlainDNSTestSuite))
}

func (suite *PlainDNSTestSuite) TestBenchmark_Run() {
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
		suite.Run(tt.name, func() {
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

			suite.Require().NoError(err, "expected no error from benchmark run")
			assertResult(suite.T(), rs)
			suite.Equal(fmt.Sprintf(tt.wantOutputTemplate, s.Addr), buf.String())
		})
	}
}

func (suite *PlainDNSTestSuite) TestBenchmark_Run_dnssec() {
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

	suite.Require().NoError(err, "expected no error from benchmark run")
	assertResult(suite.T(), rs)
	for _, r := range rs {
		suite.Equal(map[string]struct{}{"example.org.": {}}, r.AuthenticatedDomains)
	}
}

func (suite *PlainDNSTestSuite) TestBenchmark_Run_edns0() {
	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		opt := r.IsEdns0()
		if suite.NotNil(opt) {
			suite.EqualValues(1024, opt.UDPSize())
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

	suite.Require().NoError(err, "expected no error from benchmark run")
	assertResult(suite.T(), rs)
}

func (suite *PlainDNSTestSuite) TestBenchmark_Run_edns0_ednsopt() {
	testOpt := uint16(65518)
	testOptData := "test"
	testHexOptData := hex.EncodeToString([]byte(testOptData))

	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		opt := r.IsEdns0()
		if suite.NotNil(opt) {
			suite.EqualValues(1024, opt.UDPSize())
			expectedOpt := false
			for _, v := range opt.Option {
				if v.Option() == testOpt {
					if localOpt, ok := v.(*dns.EDNS0_LOCAL); ok {
						suite.Equal(testOptData, string(localOpt.Data))
						expectedOpt = true
					}
				}
			}
			suite.True(expectedOpt)
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

	suite.Require().NoError(err, "expected no error from benchmark run")
	assertResult(suite.T(), rs)
}

func (suite *PlainDNSTestSuite) TestBenchmark_Run_probability() {
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

	suite.Require().NoError(err, "expected no error from benchmark run")
	suite.Require().Len(rs, 2, "expected results from two workers")
	suite.Zero(rs[0].Counters.Total, "Run(ctx) total counter")
	suite.Zero(rs[1].Counters.Total, "Run(ctx) total counter")
}

func (suite *PlainDNSTestSuite) TestBenchmark_Run_download_external_datasource_using_http() {
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

	suite.Require().NoError(err, "expected no error from benchmark run")
	assertResult(suite.T(), rs)
}

func (suite *PlainDNSTestSuite) TestBenchmark_Run_download_external_datasource_using_http_not_available() {
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

	suite.Require().Error(err, "expected error from benchmark run")
}

func (suite *PlainDNSTestSuite) TestBenchmark_Run_download_external_datasource_using_http_wrong_response() {
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

	suite.Require().Error(err, "expected error from benchmark run")
}

func (suite *PlainDNSTestSuite) TestBenchmark_Run_duration() {
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

	suite.Require().NoError(err, "expected no error from benchmark run")
	suite.GreaterOrEqual(rs[0].Counters.Total, int64(1), "there should be atleast one execution")
}

func (suite *PlainDNSTestSuite) TestBenchmark_Run_duration_and_count_specified_at_once() {
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

	suite.Require().Error(err, "expected error from benchmark run")
}

func (suite *PlainDNSTestSuite) TestBenchmark_Run_default_count() {
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

	suite.Require().NoError(err, "expected no error from benchmark run")
	suite.Require().Len(rs, 1, "expected results from one worker")
	suite.EqualValues(1, rs[0].Counters.Total)
	suite.EqualValues(1, rs[0].Counters.Success)
}

func (suite *PlainDNSTestSuite) TestBenchmark_Run_global_ratelimit() {
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

	suite.Require().NoError(err, "expected no error from benchmark run")
	suite.Require().Len(rs, 2, "expected results from two workers")
	// assert that total queries is 5 with +-1 precision, because benchmark cancellation based on duration is not that precise
	// and one worker can start the resolution before cancelling
	suite.InDelta(int64(5), rs[0].Counters.Total+rs[1].Counters.Total, 1.0)
}

func (suite *PlainDNSTestSuite) TestBenchmark_Run_worker_ratelimit() {
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

	suite.Require().NoError(err, "expected no error from benchmark run")
	suite.Require().Len(rs, 2, "expected results from two workers")

	// assert that total queries is 10 with +-2 precision,
	// because benchmark cancellation based on duration is not that precise
	// and each worker can start the resolution before cancelling
	suite.InDelta(int64(10), rs[0].Counters.Total+rs[1].Counters.Total, 2.0)
}

func (suite *PlainDNSTestSuite) TestBenchmark_Run_error() {
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
		WriteTimeout:   100 * time.Millisecond,
		ReadTimeout:    300 * time.Millisecond,
		ConnectTimeout: 100 * time.Millisecond,
		RequestTimeout: 500 * time.Millisecond,
		Rcodes:         true,
		Recurse:        true,
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

func (suite *PlainDNSTestSuite) TestBenchmark_Run_truncated() {
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

	suite.Require().NoError(err, "expected no error from benchmark run")
	suite.Require().Len(rs, 2, "expected results from two workers")

	suite.EqualValues(2, rs[0].Counters.Total, "there should be executions")
	suite.EqualValues(2, rs[0].Counters.Truncated, "there should be truncated messages")
	suite.EqualValues(2, rs[1].Counters.Total, "there should be executions")
	suite.EqualValues(2, rs[1].Counters.Truncated, "there should be truncated messages")
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

func (suite *PlainDNSTestSuite) TestBenchmark_Requestlog() {
	requestLogPath := suite.T().TempDir() + "/requests.log"

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

	suite.Require().NoError(err, "expected no error from benchmark run")
	assertResult(suite.T(), rs)

	requestLogFile, err := os.Open(requestLogPath)
	suite.Require().NoError(err)

	requestLogs := parseRequestLogs(suite.T(), requestLogFile)

	workerIDs := map[int]int{}
	qtypes := map[string]int{}

	for _, v := range requestLogs {
		workerIDs[v.worker]++
		qtypes[v.qtype]++

		suite.Equal("example.org.", v.qname)
		suite.NotZero(v.requestid)
		suite.NotZero(v.respid)
		suite.Equal("NOERROR", v.rcode)
		suite.Equal("qr rd", v.respflags)
		suite.Equal("<nil>", v.err)
		suite.NotZero(v.duration)
	}
	suite.Equal(map[int]int{0: 2, 1: 2}, workerIDs)
	suite.Equal(map[string]int{"AAAA": 2, "A": 2}, qtypes)
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
		assert.EqualValues(t, 2, rs.Codes[0], "Run(ctx) rstats codes NOERROR, state:"+fmt.Sprint(rs.Codes))
	}

	if assert.NotNil(t, rs.Qtypes, "Run(ctx) rstats qtypes") {
		assert.EqualValues(t, 1, rs.Qtypes[dns.TypeToString[dns.TypeA]], "Run(ctx) rstats qtypes A, state:"+fmt.Sprint(rs.Codes))
		assert.EqualValues(t, 1, rs.Qtypes[dns.TypeToString[dns.TypeAAAA]], "Run(ctx) rstats qtypes AAAA, state:"+fmt.Sprint(rs.Codes))
	}

	assert.EqualValues(t, 2, rs.Counters.Total, "Run(ctx) total counter")
	assert.Zero(t, rs.Counters.IOError, "error counter")
	assert.EqualValues(t, 2, rs.Counters.Success, "Run(ctx) success counter")
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
