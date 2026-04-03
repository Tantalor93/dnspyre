package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tantalor93/dnspyre/v3/pkg/dnsbench"
)

// parseArgs creates a fresh app + benchmark and parses the given args.
func parseArgs(args []string) (dnsbench.Benchmark, []string, error) {
	var b dnsbench.Benchmark
	var fc []string
	app := newApp(&b, &fc)
	_, err := app.Parse(args)
	return b, fc, err
}

func TestParsing_Defaults(t *testing.T) {
	b, fc, err := parseArgs([]string{"example.com"})
	require.NoError(t, err)

	// string/server defaults
	assert.Equal(t, "", b.Server, "server should default to empty")

	// query types default
	assert.Equal(t, []string{dnsbench.DefaultQueryType}, b.Types)

	// numeric defaults
	assert.Equal(t, int64(0), b.Count, "count should default to 0 (unset)")
	assert.Equal(t, uint32(dnsbench.DefaultConcurrency), b.Concurrency)
	assert.Equal(t, 0, b.Rate)
	assert.Equal(t, 0, b.RateLimitWorker)
	assert.Equal(t, int64(0), b.QperConn)
	assert.Equal(t, dnsbench.DefaultProbability, b.Probability)
	assert.Equal(t, dnsbench.DefaultHistPrecision, b.HistPre)
	assert.Equal(t, uint16(0), b.Edns0)

	// boolean defaults
	assert.True(t, b.Recurse, "recurse should default to true")
	assert.True(t, b.Rcodes, "codes should default to true")
	assert.True(t, b.HistDisplay, "distribution should default to true")
	assert.True(t, b.Color, "color should default to true")
	assert.True(t, b.ProgressBar, "progress should default to true")
	assert.False(t, b.TCP)
	assert.False(t, b.DOT)
	assert.False(t, b.JSON)
	assert.False(t, b.Silent)
	assert.False(t, b.Insecure)
	assert.False(t, b.DNSSEC)
	assert.False(t, b.Cookie)
	assert.False(t, b.RequestLogEnabled)
	assert.False(t, b.SeparateWorkerConnections)

	// duration defaults
	assert.Equal(t, dnsbench.DefaultWriteTimeout, b.WriteTimeout)
	assert.Equal(t, dnsbench.DefaultReadTimeout, b.ReadTimeout)
	assert.Equal(t, dnsbench.DefaultConnectTimeout, b.ConnectTimeout)
	assert.Equal(t, dnsbench.DefaultRequestTimeout, b.RequestTimeout)
	assert.Equal(t, time.Duration(0), b.Duration)
	assert.Equal(t, time.Duration(0), b.HistMin)
	assert.Equal(t, time.Duration(0), b.HistMax)

	// string defaults
	assert.Equal(t, "", b.EdnsOpt)
	assert.Equal(t, "", b.Ecs)
	assert.Equal(t, "", b.Csv)
	assert.Equal(t, "", b.PlotDir)
	assert.Equal(t, dnsbench.DefaultPlotFormat, b.PlotFormat)
	assert.Equal(t, "", b.DohMethod)
	assert.Equal(t, "", b.DohProtocol)
	assert.Equal(t, dnsbench.DefaultRequestLogPath, b.RequestLogPath)
	assert.Equal(t, "0s", b.RequestDelay)
	assert.Equal(t, "", b.PrometheusMetricsAddr)
	assert.Equal(t, "", b.PprofAddr)

	// positional args
	assert.Equal(t, []string{"example.com"}, b.Queries)

	// fail conditions
	assert.Empty(t, fc)
}

func TestParsing_AllFlags(t *testing.T) {
	args := []string{
		"--server", "8.8.8.8",
		"--type", "AAAA",
		"--number", "10",
		"--concurrency", "4",
		"--rate-limit", "100",
		"--rate-limit-worker", "25",
		"--query-per-conn", "5",
		"--no-recurse",
		"--probability", "0.5",
		"--ednsopt", "65518:74657374",
		"--tcp",
		"--dot",
		"--write", "2s",
		"--read", "4s",
		"--connect", "500ms",
		"--request", "10s",
		"--no-codes",
		"--min", "1ms",
		"--max", "10s",
		"--precision", "3",
		"--no-distribution",
		"--csv", "/tmp/out.csv",
		"--json",
		"--silent",
		"--no-color",
		"--plot", "/tmp/plots",
		"--plotf", "png",
		"--doh-method", "get",
		"--doh-protocol", "2",
		"--insecure",
		"--duration", "30s",
		"--no-progress",
		"--fail", "ioerror",
		"--fail", "negative",
		"--dnssec",
		"--cookie",
		"--edns0", "1232",
		"--log-requests",
		"--log-requests-path", "/tmp/req.log",
		"--separate-worker-connections",
		"--request-delay", "500ms",
		"--prometheus", ":8080",
		"--pprof", ":6060",
		"example.com",
	}

	b, fc, err := parseArgs(args)
	require.NoError(t, err)

	assert.Equal(t, "8.8.8.8", b.Server)
	assert.Equal(t, []string{"AAAA"}, b.Types)
	assert.Equal(t, int64(10), b.Count)
	assert.Equal(t, uint32(4), b.Concurrency)
	assert.Equal(t, 100, b.Rate)
	assert.Equal(t, 25, b.RateLimitWorker)
	assert.Equal(t, int64(5), b.QperConn)
	assert.False(t, b.Recurse)
	assert.Equal(t, 0.5, b.Probability)
	assert.Equal(t, "65518:74657374", b.EdnsOpt)
	assert.True(t, b.TCP)
	assert.True(t, b.DOT)
	assert.Equal(t, 2*time.Second, b.WriteTimeout)
	assert.Equal(t, 4*time.Second, b.ReadTimeout)
	assert.Equal(t, 500*time.Millisecond, b.ConnectTimeout)
	assert.Equal(t, 10*time.Second, b.RequestTimeout)
	assert.False(t, b.Rcodes)
	assert.Equal(t, time.Millisecond, b.HistMin)
	assert.Equal(t, 10*time.Second, b.HistMax)
	assert.Equal(t, 3, b.HistPre)
	assert.False(t, b.HistDisplay)
	assert.Equal(t, "/tmp/out.csv", b.Csv)
	assert.True(t, b.JSON)
	assert.True(t, b.Silent)
	assert.False(t, b.Color)
	assert.Equal(t, "/tmp/plots", b.PlotDir)
	assert.Equal(t, "png", b.PlotFormat)
	assert.Equal(t, dnsbench.GetHTTPMethod, b.DohMethod)
	assert.Equal(t, dnsbench.HTTP2Proto, b.DohProtocol)
	assert.True(t, b.Insecure)
	assert.Equal(t, 30*time.Second, b.Duration)
	assert.False(t, b.ProgressBar)
	assert.True(t, b.DNSSEC)
	assert.True(t, b.Cookie)
	assert.Equal(t, uint16(1232), b.Edns0)
	assert.True(t, b.RequestLogEnabled)
	assert.Equal(t, "/tmp/req.log", b.RequestLogPath)
	assert.True(t, b.SeparateWorkerConnections)
	assert.Equal(t, "500ms", b.RequestDelay)
	assert.Equal(t, ":8080", b.PrometheusMetricsAddr)
	assert.Equal(t, ":6060", b.PprofAddr)
	assert.Equal(t, []string{"example.com"}, b.Queries)
	assert.Equal(t, []string{ioerrorFailCondition, negativeFailCondition}, fc)
}

func TestParsing_ShortFlags(t *testing.T) {
	args := []string{
		"-s", "1.1.1.1",
		"-t", "MX",
		"-n", "5",
		"-c", "8",
		"-l", "50",
		"--no-recurse",
		"-d", "1m",
		"example.org",
	}

	b, _, err := parseArgs(args)
	require.NoError(t, err)

	assert.Equal(t, "1.1.1.1", b.Server)
	assert.Equal(t, []string{"MX"}, b.Types)
	assert.Equal(t, int64(5), b.Count)
	assert.Equal(t, uint32(8), b.Concurrency)
	assert.Equal(t, 50, b.Rate)
	assert.False(t, b.Recurse)
	assert.Equal(t, time.Minute, b.Duration)
	assert.Equal(t, []string{"example.org"}, b.Queries)
}

func TestParsing_RepeatableTypeFlag(t *testing.T) {
	args := []string{
		"--type", "A",
		"--type", "AAAA",
		"--type", "MX",
		"example.com",
	}

	b, _, err := parseArgs(args)
	require.NoError(t, err)
	assert.Equal(t, []string{"A", "AAAA", "MX"}, b.Types)
}

func TestParsing_RepeatableFailFlag(t *testing.T) {
	args := []string{
		"--fail", "ioerror",
		"--fail", "negative",
		"--fail", "error",
		"--fail", "idmismatch",
		"example.com",
	}

	_, fc, err := parseArgs(args)
	require.NoError(t, err)
	assert.Equal(t, []string{
		ioerrorFailCondition,
		negativeFailCondition,
		errorFailCondition,
		idmismatchFailCondition,
	}, fc)
}

func TestParsing_MultipleQueries(t *testing.T) {
	args := []string{
		"example.com",
		"example.org",
		"example.net",
	}

	b, _, err := parseArgs(args)
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com", "example.org", "example.net"}, b.Queries)
}

func TestParsing_NoQueries(t *testing.T) {
	b, _, err := parseArgs([]string{})
	require.NoError(t, err)
	assert.Nil(t, b.Queries, "queries should be nil when not provided")
}

func TestParsing_InvalidPlotFormat(t *testing.T) {
	_, _, err := parseArgs([]string{"--plotf", "bmp", "example.com"})
	assert.Error(t, err)
}

func TestParsing_InvalidDohMethod(t *testing.T) {
	_, _, err := parseArgs([]string{"--doh-method", "put", "example.com"})
	assert.Error(t, err)
}

func TestParsing_InvalidDohProtocol(t *testing.T) {
	_, _, err := parseArgs([]string{"--doh-protocol", "4", "example.com"})
	assert.Error(t, err)
}

func TestParsing_InvalidFailCondition(t *testing.T) {
	_, _, err := parseArgs([]string{"--fail", "unknown", "example.com"})
	assert.Error(t, err)
}

func TestParsing_InvalidQueryType(t *testing.T) {
	_, _, err := parseArgs([]string{"--type", "INVALID", "example.com"})
	assert.Error(t, err)
}

func TestParsing_InvalidNumber(t *testing.T) {
	_, _, err := parseArgs([]string{"--number", "abc", "example.com"})
	assert.Error(t, err)
}

func TestParsing_InvalidConcurrency(t *testing.T) {
	_, _, err := parseArgs([]string{"--concurrency", "-1", "example.com"})
	assert.Error(t, err)
}

func TestParsing_InvalidDuration(t *testing.T) {
	_, _, err := parseArgs([]string{"--duration", "notaduration", "example.com"})
	assert.Error(t, err)
}

func TestParsing_UnknownFlag(t *testing.T) {
	_, _, err := parseArgs([]string{"--nonexistent", "example.com"})
	assert.Error(t, err)
}

func TestParsing_ValidPlotFormats(t *testing.T) {
	for _, format := range []string{"svg", "png", "jpg"} {
		t.Run(format, func(t *testing.T) {
			b, _, err := parseArgs([]string{"--plotf", format, "example.com"})
			require.NoError(t, err)
			assert.Equal(t, format, b.PlotFormat)
		})
	}
}

func TestParsing_ValidDohMethods(t *testing.T) {
	for _, method := range []string{"get", "post"} {
		t.Run(method, func(t *testing.T) {
			b, _, err := parseArgs([]string{"--doh-method", method, "example.com"})
			require.NoError(t, err)
			assert.Equal(t, method, b.DohMethod)
		})
	}
}

func TestParsing_ValidDohProtocols(t *testing.T) {
	for _, proto := range []string{"1.1", "2", "3"} {
		t.Run(proto, func(t *testing.T) {
			b, _, err := parseArgs([]string{"--doh-protocol", proto, "example.com"})
			require.NoError(t, err)
			assert.Equal(t, proto, b.DohProtocol)
		})
	}
}

func TestParsing_ValidFailConditions(t *testing.T) {
	for _, cond := range []string{"ioerror", "negative", "error", "idmismatch"} {
		t.Run(cond, func(t *testing.T) {
			_, fc, err := parseArgs([]string{"--fail", cond, "example.com"})
			require.NoError(t, err)
			assert.Equal(t, []string{cond}, fc)
		})
	}
}

func TestParsing_EcsFlag(t *testing.T) {
	b, _, err := parseArgs([]string{"--ecs", "192.0.2.0/24", "example.com"})
	require.NoError(t, err)
	assert.Equal(t, "192.0.2.0/24", b.Ecs)
}

func TestParsing_FreshBenchmarkPerParse(t *testing.T) {
	// Verify that each call to parseArgs returns a fresh benchmark with no state leakage.
	b1, _, err := parseArgs([]string{"--server", "8.8.8.8", "--concurrency", "10", "example.com"})
	require.NoError(t, err)
	assert.Equal(t, "8.8.8.8", b1.Server)
	assert.Equal(t, uint32(10), b1.Concurrency)

	b2, _, err := parseArgs([]string{"example.org"})
	require.NoError(t, err)
	assert.Equal(t, "", b2.Server, "server should be fresh empty string")
	assert.Equal(t, uint32(dnsbench.DefaultConcurrency), b2.Concurrency, "concurrency should be fresh default")
	assert.Equal(t, []string{"example.org"}, b2.Queries)
}

func TestParsing_QuerySources(t *testing.T) {
	b, _, err := parseArgs([]string{
		"google.com",
		"https://example.com/domains",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"google.com", "https://example.com/domains"}, b.Queries)
}
