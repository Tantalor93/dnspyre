package reporter_test

import (
	"bytes"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tantalor93/dnspyre/v3/pkg/dnsbench"
	"github.com/tantalor93/dnspyre/v3/pkg/reporter"
)

func Test_PrintReport(t *testing.T) {
	buffer := bytes.Buffer{}
	b, rs := testReportData(&buffer)

	err := reporter.PrintReport(&b, []*dnsbench.ResultStats{&rs}, time.Now(), time.Second)

	require.NoError(t, err)
	assert.Equal(t, readResource("successReport"), buffer.String())
}

func Test_PrintReport_dnssec(t *testing.T) {
	buffer := bytes.Buffer{}
	b, rs := testReportData(&buffer)
	b.DNSSEC = true
	rs.AuthenticatedDomains = map[string]struct{}{"example.org.": {}}

	err := reporter.PrintReport(&b, []*dnsbench.ResultStats{&rs}, time.Now(), time.Second)
	require.NoError(t, err)
	assert.Equal(t, readResource("dnssecReport"), buffer.String())
}

func Test_PrintReport_doh(t *testing.T) {
	buffer := bytes.Buffer{}
	b, rs := testReportData(&buffer)
	rs.DoHStatusCodes = map[int]int64{
		200: 2,
		500: 1,
	}

	err := reporter.PrintReport(&b, []*dnsbench.ResultStats{&rs}, time.Now(), time.Second)
	require.NoError(t, err)
	assert.Equal(t, readResource("dohReport"), buffer.String())
}

func Test_PrintReport_json(t *testing.T) {
	buffer := bytes.Buffer{}
	b, rs := testReportData(&buffer)
	b.JSON = true
	b.Rcodes = true
	b.HistDisplay = true

	err := reporter.PrintReport(&b, []*dnsbench.ResultStats{&rs}, time.Now(), time.Second)
	require.NoError(t, err)
	assert.Equal(t, readResource("jsonReport"), buffer.String())
}

func Test_PrintReport_json_dnssec(t *testing.T) {
	buffer := bytes.Buffer{}
	b, rs := testReportData(&buffer)
	b.JSON = true
	b.Rcodes = true
	b.HistDisplay = true
	b.DNSSEC = true
	rs.AuthenticatedDomains = map[string]struct{}{"example.org.": {}}

	err := reporter.PrintReport(&b, []*dnsbench.ResultStats{&rs}, time.Now(), time.Second)
	require.NoError(t, err)
	assert.Equal(t, readResource("jsonDnssecReport"), buffer.String())
}

func Test_PrintReport_json_doh(t *testing.T) {
	buffer := bytes.Buffer{}
	b, rs := testReportData(&buffer)
	b.JSON = true
	b.Rcodes = true
	b.HistDisplay = true
	rs.DoHStatusCodes = map[int]int64{
		200: 2,
	}

	err := reporter.PrintReport(&b, []*dnsbench.ResultStats{&rs}, time.Now(), time.Second)
	require.NoError(t, err)
	assert.Equal(t, readResource("jsonDohReport"), buffer.String())
}

func Test_PrintReport_errors(t *testing.T) {
	buffer := bytes.Buffer{}
	b, rs := testReportDataWithServerDNSErrors(&buffer)

	err := reporter.PrintReport(&b, []*dnsbench.ResultStats{&rs}, time.Now(), time.Second)
	require.NoError(t, err)
	assert.Equal(t, readResource("errorReport"), buffer.String())
}

func Test_PrintReport_plot(t *testing.T) {
	dir := t.TempDir()

	buffer := bytes.Buffer{}
	b, rs := testReportData(&buffer)
	b.PlotDir = dir
	b.PlotFormat = dnsbench.DefaultPlotFormat

	err := reporter.PrintReport(&b, []*dnsbench.ResultStats{&rs}, time.Now(), time.Second)

	require.NoError(t, err)

	testDir, err := os.ReadDir(dir)

	require.NoError(t, err)
	require.Len(t, testDir, 1)

	graphsDir := testDir[0].Name()
	assert.True(t, strings.HasPrefix(graphsDir, "graphs-"))

	graphsDirContent, err := os.ReadDir(filepath.Join(dir, graphsDir))
	require.NoError(t, err)

	var graphFiles []string
	for _, v := range graphsDirContent {
		graphFiles = append(graphFiles, v.Name())
	}

	assert.ElementsMatch(t, graphFiles,
		[]string{
			"errorrate-lineplot.svg", "latency-boxplot.svg", "latency-histogram.svg", "latency-lineplot.svg",
			"responses-barchart.svg", "throughput-lineplot.svg",
		},
	)
}

func Test_PrintReport_plot_error(t *testing.T) {
	dir := t.TempDir()

	buffer := bytes.Buffer{}
	b, rs := testReportData(&buffer)
	b.PlotDir = dir + "/non-existing-directory"
	b.PlotFormat = dnsbench.DefaultPlotFormat

	err := reporter.PrintReport(&b, []*dnsbench.ResultStats{&rs}, time.Now(), time.Second)

	require.Error(t, err)
}

func testReportData(testOutputWriter io.Writer) (dnsbench.Benchmark, dnsbench.ResultStats) {
	b := dnsbench.Benchmark{
		HistPre: 1,
		Writer:  testOutputWriter,
	}

	h := hdrhistogram.New(0, 0, 1)
	h.RecordValue(5)
	h.RecordValue(10)
	d1 := dnsbench.Datapoint{Duration: 5, Start: time.Unix(0, 0)}
	d2 := dnsbench.Datapoint{Duration: 10, Start: time.Unix(0, 0)}
	addr, err := net.ResolveUDPAddr("udp", "8.8.8.8:53")
	if err != nil {
		panic(err)
	}
	saddr1, err := net.ResolveUDPAddr("udp", "127.0.0.1:65359")
	if err != nil {
		panic(err)
	}
	saddr2, err := net.ResolveUDPAddr("udp", "127.0.0.1:65360")
	if err != nil {
		panic(err)
	}
	rs := dnsbench.ResultStats{
		Codes: map[int]int64{
			dns.RcodeSuccess: 2,
		},
		Qtypes: map[string]int64{
			"A": 2,
		},
		Hist:    h,
		Timings: []dnsbench.Datapoint{d1, d2},
		Counters: &dnsbench.Counters{
			Total:      1,
			IOError:    6,
			Success:    4,
			IDmismatch: 10,
			Truncated:  7,
			Negative:   8,
			Error:      9,
		},
		Errors: []dnsbench.ErrorDatapoint{
			{Start: time.Unix(0, 0), Err: errors.New("test2")},
			{Start: time.Unix(0, 0), Err: errors.New("test")},
			{Start: time.Unix(0, 0), Err: &net.OpError{Op: "read", Net: "udp", Addr: addr, Source: saddr1}},
			{Start: time.Unix(0, 0), Err: &net.OpError{Op: "read", Net: "udp", Addr: addr, Source: saddr2}},
			{Start: time.Unix(0, 0), Err: errors.New("test2")},
			{Start: time.Unix(0, 0), Err: errors.New("test2")},
		},
	}
	return b, rs
}

func testReportDataWithServerDNSErrors(testOutputWriter io.Writer) (dnsbench.Benchmark, dnsbench.ResultStats) {
	b := dnsbench.Benchmark{
		HistPre: 1,
		Writer:  testOutputWriter,
	}
	h := hdrhistogram.New(0, 0, 1)
	rs := dnsbench.ResultStats{
		Codes: map[int]int64{},
		Qtypes: map[string]int64{
			"A": 3,
		},
		Hist:    h,
		Timings: []dnsbench.Datapoint{},
		Counters: &dnsbench.Counters{
			Total:   3,
			IOError: 3,
		},
		Errors: []dnsbench.ErrorDatapoint{
			{Start: time.Unix(0, 0), Err: &net.DNSError{Err: "no such host", Name: "unknown.host.com"}},
			{Start: time.Unix(0, 0), Err: &net.DNSError{Err: "no such host", Name: "unknown.host.com"}},
			{Start: time.Unix(0, 0), Err: &net.DNSError{Err: "no such host", Name: "unknown.host.com"}},
		},
	}
	return b, rs
}

func readResource(resource string) string {
	open, err := os.Open("testdata/" + resource)
	if err != nil {
		panic(err)
	}
	all, err := io.ReadAll(open)
	if err != nil {
		panic(err)
	}
	data := string(all)
	return data
}
