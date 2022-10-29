package dnspyre

import (
	"crypto/tls"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/miekg/dns"
)

func dialConnection(b *Benchmark, m *dns.Msg, st *ResultStats) (*dns.Conn, error) {
	co, err := dial(b)
	if err != nil {
		st.Counters.ConnError++
		return nil, err
	}
	if udpSize := b.UDPSize; udpSize > 0 {
		m.SetEdns0(udpSize, true)
		co.UDPSize = udpSize
	}
	if ednsOpt := b.EdnsOpt; len(ednsOpt) > 0 {
		o := m.IsEdns0()
		if o == nil {
			m.SetEdns0(4096, true)
			o = m.IsEdns0()
		}
		s := strings.Split(ednsOpt, ":")
		data, err := hex.DecodeString(s[1])
		if err != nil {
			panic(err)
		}
		code, err := strconv.ParseUint(s[0], 10, 16)
		if err != nil {
			panic(err)
		}
		o.Option = append(o.Option, &dns.EDNS0_LOCAL{Code: uint16(code), Data: data})
	}
	return co, err
}

func dial(b *Benchmark) (*dns.Conn, error) {
	network := "udp"
	if b.TCP || b.DOT {
		network = "tcp"
	}

	if b.DOT {
		// #nosec
		return dns.DialTimeoutWithTLS(network, b.Server, &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: b.Insecure}, dnsTimeout)
	}
	return dns.DialTimeout(network, b.Server, dnsTimeout)
}
