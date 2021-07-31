package doh

import (
	"bytes"
	"context"
	"errors"
	"net/http"

	"github.com/miekg/dns"
)

// Client encapsulates and provides logic for querying DNS servers over DoH
type Client struct {
	c *http.Client
}

// NewClient creates new Client instance with standard net/http client. If nil, default http.Client is used.
func NewClient(c *http.Client) *Client {
	if c == nil {
		c = &http.Client{}
	}
	return &Client{c}
}

// PostSend sends DNS message to the given DNS server over DoH using POST, see https://datatracker.ietf.org/doc/html/rfc8484#section-4.1
func (dc *Client) PostSend(ctx context.Context, server string, msg *dns.Msg) (*dns.Msg, error) {
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

	resp, err := dc.c.Do(request)
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
