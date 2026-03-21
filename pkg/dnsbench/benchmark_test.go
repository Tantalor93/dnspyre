package dnsbench

import (
	"math/rand"
	"net"
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
			name:      "both ednsopt code 8 and ecs specified",
			benchmark: Benchmark{Server: "8.8.8.8", EdnsOpt: "8:000118005100c6", Ecs: "192.0.2.0/24"},
			wantErr:   true,
		},
		{
			name:         "ednsopt with non-ECS code and ecs can be used together",
			benchmark:    Benchmark{Server: "8.8.8.8", EdnsOpt: "10:0001", Ecs: "192.0.2.0/24"},
			assertServer: assertServerEqual("8.8.8.8:53"),
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
		{
			name:         "valid IPv4 local address",
			benchmark:    Benchmark{Server: "8.8.8.8", LocalAddr: "192.168.1.100"},
			assertServer: assertServerEqual("8.8.8.8:53"),
		},
		{
			name:         "valid IPv4 CIDR local address",
			benchmark:    Benchmark{Server: "8.8.8.8", LocalAddr: "192.168.1.0/24"},
			assertServer: assertServerEqual("8.8.8.8:53"),
		},
		{
			name:         "valid IPv6 local address",
			benchmark:    Benchmark{Server: "8.8.8.8", LocalAddr: "2001:db8::1"},
			assertServer: assertServerEqual("8.8.8.8:53"),
		},
		{
			name:         "valid IPv6 CIDR local address",
			benchmark:    Benchmark{Server: "8.8.8.8", LocalAddr: "2001:db8::/32"},
			assertServer: assertServerEqual("8.8.8.8:53"),
		},
		{
			name:      "invalid local address",
			benchmark: Benchmark{Server: "8.8.8.8", LocalAddr: "invalid"},
			wantErr:   true,
		},
		{
			name:      "invalid CIDR local address",
			benchmark: Benchmark{Server: "8.8.8.8", LocalAddr: "192.168.1.0/33"},
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

func assertServerEqual(server string) assert.ValueAssertionFunc {
	return func(t assert.TestingT, val any, i2 ...interface{}) bool {
		return assert.Equal(t, server, val, i2)
	}
}

func TestRandomIPFromCIDR(t *testing.T) {
	tests := []struct {
		name string
		cidr string
	}{
		{
			name: "IPv4 /24 network",
			cidr: "192.168.1.0/24",
		},
		{
			name: "IPv4 /28 network",
			cidr: "10.0.0.0/28",
		},
		{
			name: "IPv4 /32 network (single IP)",
			cidr: "192.168.1.100/32",
		},
		{
			name: "IPv6 /64 network",
			cidr: "2001:db8::/64",
		},
		{
			name: "IPv6 /128 network (single IP)",
			cidr: "2001:db8::1/128",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ipnet, err := net.ParseCIDR(tt.cidr)
			require.NoError(t, err)

			// Generate multiple random IPs and verify they're all within the network
			// nolint:gosec
			rnd := rand.New(rand.NewSource(42))
			for i := 0; i < 100; i++ {
				randomIP := randomIPFromCIDR(ipnet, rnd)

				// Verify the generated IP is within the network
				assert.True(t, ipnet.Contains(randomIP),
					"Generated IP %s should be within network %s", randomIP, tt.cidr)
			}
		})
	}
}

func TestParseLocalAddr(t *testing.T) {
	tests := []struct {
		name             string
		localAddr        string
		wantErr          bool
		wantLocalAddrIP  string
		wantLocalAddrNet string
	}{
		{
			name:            "IPv4 address",
			localAddr:       "192.168.1.100",
			wantLocalAddrIP: "192.168.1.100",
		},
		{
			name:             "IPv4 CIDR",
			localAddr:        "192.168.1.0/24",
			wantLocalAddrIP:  "192.168.1.0",
			wantLocalAddrNet: "192.168.1.0/24",
		},
		{
			name:            "IPv6 address",
			localAddr:       "2001:db8::1",
			wantLocalAddrIP: "2001:db8::1",
		},
		{
			name:             "IPv6 CIDR",
			localAddr:        "2001:db8::/32",
			wantLocalAddrIP:  "2001:db8::",
			wantLocalAddrNet: "2001:db8::/32",
		},
		{
			name:      "invalid address",
			localAddr: "invalid",
			wantErr:   true,
		},
		{
			name:      "invalid CIDR",
			localAddr: "192.168.1.0/33",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Benchmark{
				Server:    "8.8.8.8",
				LocalAddr: tt.localAddr,
			}

			err := b.parseLocalAddr()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantLocalAddrIP, b.localAddrIP.String())

				if tt.wantLocalAddrNet != "" {
					require.NotNil(t, b.localAddrNet)
					assert.Equal(t, tt.wantLocalAddrNet, b.localAddrNet.String())
				} else {
					assert.Nil(t, b.localAddrNet)
				}
			}
		})
	}
}
