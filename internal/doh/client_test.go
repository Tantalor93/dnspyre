package doh

import (
	"context"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func Test_PostSend(t *testing.T) {
	type args struct {
		server string
		msg    *dns.Msg
	}
	tests := []struct {
		name      string
		args      args
		wantRcode int
		wantErr   bool
	}{
		{
			name:      "NOERROR DNS resolution",
			args:      args{server: "https://1.1.1.1", msg: question("google.com.")},
			wantRcode: dns.RcodeSuccess,
		},
		{
			name:      "NXDOMAIN DNS resolution ",
			args:      args{server: "https://1.1.1.1", msg: question("nxdomain.cz.")},
			wantRcode: dns.RcodeNameError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(nil)

			got, err := client.PostSend(context.Background(), tt.args.server, tt.args.msg)

			if tt.wantErr {
				assert.Error(t, err, "PostSend() error")
			} else {
				assert.NotNil(t, got, "PostSend() response")
				assert.Equal(t, tt.wantRcode, got.Rcode, "PostSend() rcode")
			}
		})
	}
}

func question(fqdn string) *dns.Msg {
	q := dns.Msg{}
	return q.SetQuestion(fqdn, dns.TypeA)
}
