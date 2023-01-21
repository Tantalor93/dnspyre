package dnspyre

import (
	"net"

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
func NewServer(network string, f dns.HandlerFunc) *Server {
	ch := make(chan bool)
	s := &dns.Server{}
	s.Handler = f

	for i := 0; i < 10; i++ {
		s.Listener, _ = net.Listen("tcp", "127.0.0.1:0")
		if network == "udp" {
			if s.Listener == nil {
				continue
			}
			s.PacketConn, _ = net.ListenPacket("udp", s.Listener.Addr().String())
			if s.PacketConn != nil {
				break
			}
		}
		if s.Listener != nil {
			break
		}
	}
	if s.Listener == nil {
		panic("failed to create new client")
	}

	s.NotifyStartedFunc = func() { close(ch) }
	go func() {
		s.ActivateAndServe()
	}()

	<-ch
	return &Server{inner: s, Addr: s.Listener.Addr().String()}
}
