package dnsbench

import (
	"fmt"
	"log"
	"time"

	"github.com/miekg/dns"
)

func logRequest(workerID uint32, req dns.Msg, resp *dns.Msg, err error, dur time.Duration) {
	rcode := "<nil>"
	respid := "<nil>"
	respflags := "<nil>"
	if resp != nil {
		rcode = dns.RcodeToString[resp.Rcode]
		respid = fmt.Sprint(resp.Id)
		respflags = getFlags(resp)
	}
	log.Printf("worker:[%v] reqid:[%d] qname:[%s] qtype:[%s] respid:[%s] rcode:[%s] respflags:[%s] err:[%v] duration:[%v]",
		workerID, req.Id, req.Question[0].Name, dns.TypeToString[req.Question[0].Qtype], respid, rcode, respflags, err, dur)
}

func getFlags(resp *dns.Msg) string {
	respflags := ""
	if resp.Response {
		respflags += "qr"
	}
	if resp.Authoritative {
		respflags += " aa"
	}
	if resp.Truncated {
		respflags += " tc"
	}
	if resp.RecursionDesired {
		respflags += " rd"
	}
	if resp.RecursionAvailable {
		respflags += " ra"
	}
	if resp.Zero {
		respflags += " z"
	}
	if resp.AuthenticatedData {
		respflags += " ad"
	}
	if resp.CheckingDisabled {
		respflags += " cd"
	}
	return respflags
}
