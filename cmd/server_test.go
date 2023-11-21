package cmd

import (
	"crypto/tls"

	"github.com/miekg/dns"
)

// Server represents simple DNS server.
type Server struct {
	Addr  string
	inner *dns.Server
}

// Close shuts down running DNS server instance.
func (s *Server) Close() {
	s.inner.Shutdown()
}

// NewServer creates and starts new DNS server instance.
func NewServer(network string, tlsConfig *tls.Config, f dns.HandlerFunc) *Server {
	ch := make(chan bool)
	s := &dns.Server{Net: network, Addr: "127.0.0.1:0", TLSConfig: tlsConfig, NotifyStartedFunc: func() { close(ch) }, Handler: f}

	go func() {
		if err := s.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	<-ch
	server := Server{inner: s}
	if network == udpNetwork {
		server.Addr = s.PacketConn.LocalAddr().String()
	} else {
		server.Addr = s.Listener.Addr().String()
	}
	return &server
}
