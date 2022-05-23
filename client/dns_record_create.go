package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// SRV defines the metata required for creating an SRV type DNS Record.
type SRV struct {
	Port     int64  `json:"port"`
	Priority int64  `json:"priority"`
	Target   string `json:"target"`
	Weight   int64  `json:"weight"`
}

// CreateDNSRecordRequest defines the information necessary to create a DNS record within Vercel.
type CreateDNSRecordRequest struct {
	Domain     string `json:"-"`
	MXPriority int64  `json:"mxPriority,omitempty"`
	Name       string `json:"name"`
	SRV        *SRV   `json:"srv,omitempty"`
	TTL        int64  `json:"ttl,omitempty"`
	Type       string `json:"type"`
	Value      string `json:"value,omitempty"`
}

// CreateProjectDomain creates a DNS record for a specified domain name within Vercel.
func (c *Client) CreateDNSRecord(ctx context.Context, teamID string, request CreateDNSRecordRequest) (r DNSRecord, err error) {
	url := fmt.Sprintf("%s/v4/domains/%s/records", c.baseURL, request.Domain)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		url,
		strings.NewReader(string(mustMarshal(request))),
	)
	if err != nil {
		return r, err
	}

	var response struct {
		RecordID string `json:"uid"`
	}
	err = c.doRequest(req, &response)
	if err != nil {
		return r, err
	}

	return c.GetDNSRecord(ctx, response.RecordID, teamID)
}
