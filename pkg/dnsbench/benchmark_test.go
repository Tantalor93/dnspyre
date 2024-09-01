package dnsbench

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBenchmark_init(t *testing.T) {
	tests := []struct {
		name                  string
		benchmark             Benchmark
		wantServer            string
		wantRequestLogPath    string
		wantErr               bool
		wantRequestDelayStart time.Duration
		wantRequestDelayEnd   time.Duration
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
		{
			name:               "request log - default path",
			benchmark:          Benchmark{Server: "8.8.8.8", RequestLogEnabled: true},
			wantServer:         "8.8.8.8:53",
			wantRequestLogPath: DefaultRequestLogPath,
		},
		{
			name:                  "constant delay",
			benchmark:             Benchmark{Server: "8.8.8.8", RequestDelay: "2s"},
			wantServer:            "8.8.8.8:53",
			wantRequestDelayStart: 2 * time.Second,
		},
		{
			name:                  "random delay",
			benchmark:             Benchmark{Server: "8.8.8.8", RequestDelay: "2s-3s"},
			wantServer:            "8.8.8.8:53",
			wantRequestDelayStart: 2 * time.Second,
			wantRequestDelayEnd:   3 * time.Second,
		},
		{
			name:      "invalid delay",
			benchmark: Benchmark{Server: "8.8.8.8", RequestDelay: "invalid"},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.benchmark.init()

			require.Equal(t, tt.wantErr, err != nil)
			if !tt.wantErr {
				assert.Equal(t, tt.wantServer, tt.benchmark.Server)
				assert.Equal(t, tt.wantRequestLogPath, tt.benchmark.RequestLogPath)
				assert.Equal(t, tt.wantRequestDelayStart, tt.benchmark.requestDelayStart)
				assert.Equal(t, tt.wantRequestDelayEnd, tt.benchmark.requestDelayEnd)
			}
		})
	}
}
