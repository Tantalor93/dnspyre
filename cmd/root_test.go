package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tantalor93/dnspyre/v3/pkg/dnsbench"
)

// parseArgs resets all package-level state and parses the given args through pApp.
// It must be called at the start of every flag-parsing test to prevent state leaking
// between table-test cases, since pApp and benchmark are package-level variables.
func parseArgs(t *testing.T, args []string) {
	t.Helper()
	benchmark = dnsbench.Benchmark{}
	failConditions = nil
	_, err := pApp.Parse(preprocessArgs(args))
	require.NoError(t, err)
}

// defaultBenchmark returns a Benchmark with all defaults that kingpin applies when
// no flags other than the positional queries argument are provided. Each test case
// should start from this value and modify only the field(s) its flag is expected to set.
func defaultBenchmark(queries []string) dnsbench.Benchmark {
	return dnsbench.Benchmark{
		Types:          []string{dnsbench.DefaultQueryType},
		Concurrency:    uint32(dnsbench.DefaultConcurrency),
		Recurse:        true,
		Probability:    dnsbench.DefaultProbability,
		WriteTimeout:   dnsbench.DefaultWriteTimeout,
		ReadTimeout:    dnsbench.DefaultReadTimeout,
		ConnectTimeout: dnsbench.DefaultConnectTimeout,
		RequestTimeout: dnsbench.DefaultRequestTimeout,
		Rcodes:         true,
		HistPre:        dnsbench.DefaultHistPrecision,
		HistDisplay:    true,
		Color:          true,
		PlotFormat:     dnsbench.DefaultPlotFormat,
		ProgressBar:    true,
		Queries:        queries,
		RequestLogPath: dnsbench.DefaultRequestLogPath,
		RequestDelay:   "0s",
	}
}

func TestFlagParsing(t *testing.T) {
	tests := []struct {
		name                   string
		args                   []string
		expected               dnsbench.Benchmark
		expectedFailConditions []string
	}{
		// --- defaults ---
		{
			name:     "defaults are applied when no flags given",
			args:     []string{"google.com"},
			expected: defaultBenchmark([]string{"google.com"}),
		},

		// --- --edns0 special behaviour ---
		{
			name:     "edns0 not provided keeps 0",
			args:     []string{"google.com"},
			expected: defaultBenchmark([]string{"google.com"}),
		},
		{
			name:     "edns0=0 explicitly provided disables edns0",
			args:     []string{"--edns0=0", "google.com"},
			expected: defaultBenchmark([]string{"google.com"}),
		},
		{
			name: "edns0=0 explicitly provided uses DefaultEdns0BufferSize",
			args: []string{"--edns0=0", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				return b
			}(),
		},
		{
			name: "edns0=1232 explicit value is preserved",
			args: []string{"--edns0=1232", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.Edns0 = 1232
				return b
			}(),
		},
		{
			name: "edns0=4096 explicit value is preserved",
			args: []string{"--edns0=4096", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.Edns0 = 4096
				return b
			}(),
		},

		// --- individual flags ---
		{
			name: "server flag long form",
			args: []string{"--server=8.8.8.8", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.Server = "8.8.8.8"
				return b
			}(),
		},
		{
			name: "server flag short form",
			args: []string{"-s", "8.8.8.8", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.Server = "8.8.8.8"
				return b
			}(),
		},
		{
			name: "concurrency flag",
			args: []string{"--concurrency=10", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.Concurrency = 10
				return b
			}(),
		},
		{
			name: "number flag long form",
			args: []string{"--number=100", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.Count = 100
				return b
			}(),
		},
		{
			name: "number flag short form",
			args: []string{"-n", "100", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.Count = 100
				return b
			}(),
		},
		{
			name: "duration flag",
			args: []string{"--duration=30s", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.Duration = 30 * time.Second
				return b
			}(),
		},
		{
			name: "type flag single",
			args: []string{"--type=AAAA", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.Types = []string{"AAAA"}
				return b
			}(),
		},
		{
			name: "type flag multiple",
			args: []string{"--type=A", "--type=AAAA", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.Types = []string{"A", "AAAA"}
				return b
			}(),
		},
		{
			name: "recurse disabled",
			args: []string{"--no-recurse", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.Recurse = false
				return b
			}(),
		},
		{
			name: "dnssec flag",
			args: []string{"--dnssec", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.DNSSEC = true
				return b
			}(),
		},
		{
			name: "tcp flag",
			args: []string{"--tcp", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.TCP = true
				return b
			}(),
		},
		{
			name: "dot flag",
			args: []string{"--dot", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.DOT = true
				return b
			}(),
		},
		{
			name: "insecure flag",
			args: []string{"--insecure", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.Insecure = true
				return b
			}(),
		},
		{
			name: "cookie flag",
			args: []string{"--cookie", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.Cookie = true
				return b
			}(),
		},
		{
			name: "ecs flag",
			args: []string{"--ecs=192.0.2.0/24", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.Ecs = "192.0.2.0/24"
				return b
			}(),
		},
		{
			name: "ednsopt flag",
			args: []string{"--ednsopt=10:AABB", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.EdnsOpt = "10:AABB"
				return b
			}(),
		},
		{
			name: "rate-limit flag",
			args: []string{"--rate-limit=500", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.Rate = 500
				return b
			}(),
		},
		{
			name: "rate-limit-worker flag",
			args: []string{"--rate-limit-worker=50", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.RateLimitWorker = 50
				return b
			}(),
		},
		{
			name: "query-per-conn flag",
			args: []string{"--query-per-conn=100", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.QperConn = 100
				return b
			}(),
		},
		{
			name: "probability flag",
			args: []string{"--probability=0.5", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.Probability = 0.5
				return b
			}(),
		},
		{
			name: "doh-method get",
			args: []string{"--doh-method=get", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.DohMethod = dnsbench.GetHTTPMethod
				return b
			}(),
		},
		{
			name: "doh-protocol 2",
			args: []string{"--doh-protocol=2", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.DohProtocol = dnsbench.HTTP2Proto
				return b
			}(),
		},
		{
			name:                   "fail flag single condition",
			args:                   []string{"--fail=ioerror", "google.com"},
			expected:               defaultBenchmark([]string{"google.com"}),
			expectedFailConditions: []string{"ioerror"},
		},
		{
			name:                   "fail flag multiple conditions",
			args:                   []string{"--fail=ioerror", "--fail=negative", "--fail=error", "google.com"},
			expected:               defaultBenchmark([]string{"google.com"}),
			expectedFailConditions: []string{"ioerror", "negative", "error"},
		},
		{
			name: "separate-worker-connections flag",
			args: []string{"--separate-worker-connections", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.SeparateWorkerConnections = true
				return b
			}(),
		},
		{
			name: "request-delay constant value",
			args: []string{"--request-delay=500ms", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.RequestDelay = "500ms"
				return b
			}(),
		},
		{
			name: "request-delay randomized interval",
			args: []string{"--request-delay=100ms-500ms", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.RequestDelay = "100ms-500ms"
				return b
			}(),
		},
		{
			name: "silent flag",
			args: []string{"--silent", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.Silent = true
				return b
			}(),
		},
		{
			name: "json flag",
			args: []string{"--json", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.JSON = true
				return b
			}(),
		},
		{
			name: "write timeout flag",
			args: []string{"--write=2s", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.WriteTimeout = 2 * time.Second
				return b
			}(),
		},
		{
			name: "read timeout flag",
			args: []string{"--read=5s", "google.com"},
			expected: func() dnsbench.Benchmark {
				b := defaultBenchmark([]string{"google.com"})
				b.ReadTimeout = 5 * time.Second
				return b
			}(),
		},
		{
			name:     "queries positional argument",
			args:     []string{"google.com", "cloudflare.com"},
			expected: defaultBenchmark([]string{"google.com", "cloudflare.com"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseArgs(t, tt.args)
			assert.Equal(t, tt.expected, benchmark)
			assert.Equal(t, tt.expectedFailConditions, failConditions)
		})
	}
}
