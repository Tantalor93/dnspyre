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
		assertServer          assert.ValueAssertionFunc
		wantRequestLogPath    string
		wantErr               bool
		wantRequestDelayStart time.Duration
		wantRequestDelayEnd   time.Duration
	}{
		{
			name:         "server - IPv4",
			benchmark:    Benchmark{Server: "8.8.8.8"},
			assertServer: assertServerEqual("8.8.8.8:53"),
		},
		{
			name:         "server - IPv4 with port",
			benchmark:    Benchmark{Server: "8.8.8.8:53"},
			assertServer: assertServerEqual("8.8.8.8:53"),
		},
		{
			name:         "server - IPv6",
			benchmark:    Benchmark{Server: "fddd:dddd::"},
			assertServer: assertServerEqual("[fddd:dddd::]:53"),
		},
		{
			name:         "server - IPv6",
			benchmark:    Benchmark{Server: "fddd:dddd::"},
			assertServer: assertServerEqual("[fddd:dddd::]:53"),
		},
		{
			name:         "server - IPv6 with port",
			benchmark:    Benchmark{Server: "fddd:dddd::"},
			assertServer: assertServerEqual("[fddd:dddd::]:53"),
		},
		{
			name:         "server - DoT with IP address",
			benchmark:    Benchmark{Server: "8.8.8.8", DOT: true},
			assertServer: assertServerEqual("8.8.8.8:853"),
		},
		{
			name:         "server - HTTPS url",
			benchmark:    Benchmark{Server: "https://1.1.1.1"},
			assertServer: assertServerEqual("https://1.1.1.1/dns-query"),
		},
		{
			name:         "server - HTTP url",
			benchmark:    Benchmark{Server: "http://127.0.0.1"},
			assertServer: assertServerEqual("http://127.0.0.1/dns-query"),
		},
		{
			name:         "server - custom HTTP url",
			benchmark:    Benchmark{Server: "http://127.0.0.1/custom/dns-query"},
			assertServer: assertServerEqual("http://127.0.0.1/custom/dns-query"),
		},
		{
			name:         "server - QUIC url",
			benchmark:    Benchmark{Server: "quic://dns.adguard-dns.com"},
			assertServer: assertServerEqual("dns.adguard-dns.com:853"),
		},
		{
			name:         "server - QUIC url with port",
			benchmark:    Benchmark{Server: "quic://localhost:853"},
			assertServer: assertServerEqual("localhost:853"),
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
			name:         "Missing server",
			benchmark:    Benchmark{},
			assertServer: assert.NotEmpty,
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
			name:         "valid IPv4 ECS",
			benchmark:    Benchmark{Server: "8.8.8.8", Ecs: "192.0.2.0/24"},
			assertServer: assertServerEqual("8.8.8.8:53"),
		},
		{
			name:         "valid IPv6 ECS",
			benchmark:    Benchmark{Server: "8.8.8.8", Ecs: "2001:db8::/32"},
			assertServer: assertServerEqual("8.8.8.8:53"),
		},
		{
			name:      "invalid ECS format",
			benchmark: Benchmark{Server: "8.8.8.8", Ecs: "invalid"},
			wantErr:   true,
		},
		{
			name:      "invalid ECS CIDR",
			benchmark: Benchmark{Server: "8.8.8.8", Ecs: "192.0.2.256/24"},
			wantErr:   true,
		},
		{
			name:      "both ednsopt and ecs specified",
			benchmark: Benchmark{Server: "8.8.8.8", EdnsOpt: "8:000118005100c6", Ecs: "192.0.2.0/24"},
			wantErr:   true,
		},
		{
			name:               "request log - default path",
			benchmark:          Benchmark{Server: "8.8.8.8", RequestLogEnabled: true},
			assertServer:       assertServerEqual("8.8.8.8:53"),
			wantRequestLogPath: DefaultRequestLogPath,
		},
		{
			name:                  "constant delay",
			benchmark:             Benchmark{Server: "8.8.8.8", RequestDelay: "2s"},
			assertServer:          assertServerEqual("8.8.8.8:53"),
			wantRequestDelayStart: 2 * time.Second,
		},
		{
			name:                  "random delay",
			benchmark:             Benchmark{Server: "8.8.8.8", RequestDelay: "2s-3s"},
			assertServer:          assertServerEqual("8.8.8.8:53"),
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
				tt.assertServer(t, tt.benchmark.Server)
				assert.Equal(t, tt.wantRequestLogPath, tt.benchmark.RequestLogPath)
				assert.Equal(t, tt.wantRequestDelayStart, tt.benchmark.requestDelayStart)
				assert.Equal(t, tt.wantRequestDelayEnd, tt.benchmark.requestDelayEnd)
			}
		})
	}
}

func TestParseECS(t *testing.T) {
	tests := []struct {
		name         string
		cidr         string
		wantFamily   uint16
		wantNetmask  uint8
		wantAddress  string
		wantErr      bool
	}{
		{
			name:        "valid IPv4 /24",
			cidr:        "192.0.2.0/24",
			wantFamily:  1,
			wantNetmask: 24,
			wantAddress: "192.0.2.0",
		},
		{
			name:        "valid IPv4 /22",
			cidr:        "204.15.220.0/22",
			wantFamily:  1,
			wantNetmask: 22,
			wantAddress: "204.15.220.0",
		},
		{
			name:        "valid IPv4 /32",
			cidr:        "8.8.8.8/32",
			wantFamily:  1,
			wantNetmask: 32,
			wantAddress: "8.8.8.8",
		},
		{
			name:        "valid IPv6 /32",
			cidr:        "2001:db8::/32",
			wantFamily:  2,
			wantNetmask: 32,
			wantAddress: "2001:db8::",
		},
		{
			name:        "valid IPv6 /64",
			cidr:        "2001:db8:abcd::/64",
			wantFamily:  2,
			wantNetmask: 64,
			wantAddress: "2001:db8:abcd::",
		},
		{
			name:        "valid IPv6 /128",
			cidr:        "2001:db8::1/128",
			wantFamily:  2,
			wantNetmask: 128,
			wantAddress: "2001:db8::1",
		},
		{
			name:    "invalid CIDR format",
			cidr:    "invalid",
			wantErr: true,
		},
		{
			name:    "invalid IP address",
			cidr:    "999.999.999.999/24",
			wantErr: true,
		},
		{
			name:    "missing netmask",
			cidr:    "192.0.2.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subnet, err := parseECS(tt.cidr)
			
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantFamily, subnet.Family)
			assert.Equal(t, tt.wantNetmask, subnet.SourceNetmask)
			assert.Equal(t, uint8(0), subnet.SourceScope)
			assert.Equal(t, tt.wantAddress, subnet.Address.String())
		})
	}
}

func assertServerEqual(server string) assert.ValueAssertionFunc {
	return func(t assert.TestingT, val any, i2 ...interface{}) bool {
		return assert.Equal(t, server, val, i2)
	}
}
