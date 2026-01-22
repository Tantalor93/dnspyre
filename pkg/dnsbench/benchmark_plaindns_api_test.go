package dnsbench_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/miekg/dns"
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

func (suite *PlainDNSTestSuite) TestBenchmark_Run_ecs() {
	testCIDR := "192.0.2.0/24"

	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		opt := r.IsEdns0()
		if suite.NotNil(opt) {
			expectedECS := false
			for _, v := range opt.Option {
				if v.Option() == dns.EDNS0SUBNET {
					if subnetOpt, ok := v.(*dns.EDNS0_SUBNET); ok {
						suite.Equal(uint16(1), subnetOpt.Family) // IPv4
						suite.Equal(uint8(24), subnetOpt.SourceNetmask)
						suite.Equal(uint8(0), subnetOpt.SourceScope)
						suite.Equal("192.0.2.0", subnetOpt.Address.String())
						expectedECS = true
					}
				}
			}
			suite.True(expectedECS, "ECS option should be present in the request")
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
		Ecs:            testCIDR,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	suite.Require().NoError(err, "expected no error from benchmark run")
	assertResult(suite.T(), rs)
}

func (suite *PlainDNSTestSuite) TestBenchmark_Run_ecs_ipv6() {
	testCIDR := "2001:db8::/32"

	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		opt := r.IsEdns0()
		if suite.NotNil(opt) {
			expectedECS := false
			for _, v := range opt.Option {
				if v.Option() == dns.EDNS0SUBNET {
					if subnetOpt, ok := v.(*dns.EDNS0_SUBNET); ok {
						suite.Equal(uint16(2), subnetOpt.Family) // IPv6
						suite.Equal(uint8(32), subnetOpt.SourceNetmask)
						suite.Equal(uint8(0), subnetOpt.SourceScope)
						suite.Equal("2001:db8::", subnetOpt.Address.String())
						expectedECS = true
					}
				}
			}
			suite.True(expectedECS, "ECS option should be present in the request")
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
		Ecs:            testCIDR,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	suite.Require().NoError(err, "expected no error from benchmark run")
	assertResult(suite.T(), rs)
}

func (suite *PlainDNSTestSuite) TestBenchmark_Run_ecs_and_ednsopt() {
	testCIDR := "192.0.2.0/24"
	testOpt := uint16(65518)
	testOptData := "test"
	testHexOptData := hex.EncodeToString([]byte(testOptData))

	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		opt := r.IsEdns0()
		if suite.NotNil(opt) {
			expectedECS := false
			expectedCustomOpt := false

			for _, v := range opt.Option {
				if v.Option() == dns.EDNS0SUBNET {
					if subnetOpt, ok := v.(*dns.EDNS0_SUBNET); ok {
						suite.Equal(uint16(1), subnetOpt.Family) // IPv4
						suite.Equal(uint8(24), subnetOpt.SourceNetmask)
						suite.Equal("192.0.2.0", subnetOpt.Address.String())
						expectedECS = true
					}
				}
				if v.Option() == testOpt {
					if localOpt, ok := v.(*dns.EDNS0_LOCAL); ok {
						suite.Equal(testOptData, string(localOpt.Data))
						expectedCustomOpt = true
					}
				}
			}

			suite.True(expectedECS, "ECS option should be present in the request")
			suite.True(expectedCustomOpt, "Custom EDNS option should be present in the request")
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
		Ecs:            testCIDR,
		EdnsOpt:        strconv.Itoa(int(testOpt)) + ":" + testHexOptData,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	suite.Require().NoError(err, "expected no error from benchmark run")
	assertResult(suite.T(), rs)
}

func (suite *PlainDNSTestSuite) TestBenchmark_Run_cookie() {
	testCookie := "24a5ac0123456789" // 8-byte client cookie

	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		opt := r.IsEdns0()
		if suite.NotNil(opt) {
			expectedCookie := false
			for _, v := range opt.Option {
				if v.Option() == dns.EDNS0COOKIE {
					if cookieOpt, ok := v.(*dns.EDNS0_COOKIE); ok {
						suite.Equal(testCookie, cookieOpt.Cookie)
						expectedCookie = true
					}
				}
			}
			suite.True(expectedCookie, "Cookie option should be present in the request")
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
		Cookie:         testCookie,
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

	buf := bytes.Buffer{}
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
		Writer:         &buf,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	suite.Require().NoError(err, "expected no error from benchmark run")
	suite.Require().Len(rs, 2, "expected results from two workers")
	// assert that total queries is 5 with +-1 precision, because benchmark cancellation based on duration is not that precise
	// and one worker can start the resolution before cancelling
	suite.InDelta(int64(5), rs[0].Counters.Total+rs[1].Counters.Total, 1.0)
	suite.Equal(
		fmt.Sprintf("Using 1 hostnames\nBenchmarking %s via udp with 2 concurrent requests (limited to 1 QPS overall)\n",
			s.Addr), buf.String(),
	)
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

	buf := bytes.Buffer{}
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
		Writer:          &buf,
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
	suite.Equal(
		fmt.Sprintf("Using 1 hostnames\nBenchmarking %s via udp with 2 concurrent requests (limited to 1 QPS per concurrent worker)\n",
			s.Addr), buf.String(),
	)
}

func (suite *PlainDNSTestSuite) TestBenchmark_Run_global_ratelimit_precendence() {
	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

		// wait some time to actually have some observable duration
		time.Sleep(time.Millisecond * 100)

		w.WriteMsg(ret)
	})
	defer s.Close()

	buf := bytes.Buffer{}
	bench := dnsbench.Benchmark{
		Queries:         []string{"example.org"},
		Types:           []string{"A"},
		Server:          s.Addr,
		TCP:             false,
		Concurrency:     2,
		Duration:        5 * time.Second,
		RateLimitWorker: 2,
		Rate:            1,
		Probability:     1,
		WriteTimeout:    1 * time.Second,
		ReadTimeout:     3 * time.Second,
		ConnectTimeout:  1 * time.Second,
		RequestTimeout:  5 * time.Second,
		Rcodes:          true,
		Recurse:         true,
		Writer:          &buf,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rs, err := bench.Run(ctx)

	suite.Require().NoError(err, "expected no error from benchmark run")
	suite.Require().Len(rs, 2, "expected results from two workers")

	// assert that total queries is 10 with +-2 precision,
	// because benchmark cancellation based on duration is not that precise
	// and each worker can start the resolution before cancelling
	suite.InDelta(int64(5), rs[0].Counters.Total+rs[1].Counters.Total, 1.0)
	suite.Equal(
		fmt.Sprintf("Using 1 hostnames\nBenchmarking %s via udp with 2 concurrent requests (limited to 1 QPS overall and 2 QPS per concurrent worker)\n",
			s.Addr), buf.String(),
	)
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

	assertRequestLogStructure(suite.T(), requestLogFile)
}

func (suite *PlainDNSTestSuite) TestBenchmark_ConstantRequestDelay() {
	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

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
		RequestDelay:   "1s",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	rs, err := bench.Run(ctx)
	benchDuration := time.Since(start)

	suite.Require().NoError(err, "expected no error from benchmark run")
	assertResult(suite.T(), rs)
	suite.InDelta(2*time.Second, benchDuration, float64(100*time.Millisecond))
}

func (suite *PlainDNSTestSuite) TestBenchmark_RandomRequestDelay() {
	s := NewServer(dnsbench.UDPTransport, nil, func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, A("example.org. IN A 127.0.0.1"))

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
		RequestDelay:   "1s-2s",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	rs, err := bench.Run(ctx)
	benchDuration := time.Since(start)

	suite.Require().NoError(err, "expected no error from benchmark run")
	assertResult(suite.T(), rs)
	suite.InDelta(4*time.Second, benchDuration, float64(2*time.Second))
}
