package doh

import (
	"bytes"
	"context"
	"errors"
	"net/http"

	"github.com/miekg/dns"
)

var client = http.Client{}

// Send sends DNS message to the given DNS server over DoH
func Send(ctx context.Context, server string, msg *dns.Msg) (*dns.Msg, error) {
	pack, err := msg.Pack()
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", server+"/dns-query", bytes.NewReader(pack))
	if err != nil {
		return nil, err
	}
	request = request.WithContext(ctx)
	request.Header.Set("Accept", "application/dns-message")
	request.Header.Set("content-type", "application/dns-message")

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected HTTP status")
	}

	buffer := bytes.Buffer{}
	_, err = buffer.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}
	res := dns.Msg{}
	err = res.Unpack(buffer.Bytes())
	if err != nil {
		return nil, err
	}
	return &res, nil
}
