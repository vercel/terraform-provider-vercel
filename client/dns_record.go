package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	Comment    string `json:"comment"`
}

// CreateDNSRecord creates a DNS record for a specified domain name within Vercel.
func (c *Client) CreateDNSRecord(ctx context.Context, teamID string, request CreateDNSRecordRequest) (r DNSRecord, err error) {
	url := fmt.Sprintf("%s/v4/domains/%s/records", c.baseURL, request.Domain)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	var response struct {
		RecordID string `json:"uid"`
	}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   string(mustMarshal(request)),
	}, &response)
	if err != nil {
		return r, err
	}

	return c.GetDNSRecord(ctx, response.RecordID, teamID)
}

// DeleteDNSRecord removes a DNS domain from Vercel.
func (c *Client) DeleteDNSRecord(ctx context.Context, domain, recordID, teamID string) error {
	url := fmt.Sprintf("%s/v2/domains/%s/records/%s", c.baseURL, domain, recordID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   "",
	}, nil)
}

// DNSRecord is the information Vercel surfaces about a DNS record associated with a particular domain.
type DNSRecord struct {
	Creator    string `json:"creator"`
	Domain     string `json:"domain"`
	ID         string `json:"id"`
	TeamID     string `json:"-"`
	Name       string `json:"name"`
	TTL        int64  `json:"ttl"`
	Value      string `json:"value"`
	RecordType string `json:"recordType"`
	Priority   int64  `json:"priority"`
	Comment    string `json:"comment"`
}

// GetDNSRecord retrieves information about a DNS domain from Vercel.
func (c *Client) GetDNSRecord(ctx context.Context, recordID, teamID string) (r DNSRecord, err error) {
	url := fmt.Sprintf("%s/domains/records/%s", c.baseURL, recordID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &r)
	r.TeamID = c.teamID(teamID)
	return r, err
}

// ListDNSRecords is a test helper for listing DNS records that exist for a given domain.
// We limit this to 100, as this is the largest limit allowed by the API.
// This is only used by the sweeper script, so this is safe to do so, but converting
// into a production ready function would require some refactoring.
func (c *Client) ListDNSRecords(ctx context.Context, domain, teamID string) (r []DNSRecord, err error) {
	url := fmt.Sprintf("%s/v4/domains/%s/records?limit=100", c.baseURL, domain)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s&teamId=%s", url, c.teamID(teamID))
	}

	dr := struct {
		Records []DNSRecord `json:"records"`
	}{}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &dr)
	for i := 0; i < len(dr.Records); i++ {
		dr.Records[i].TeamID = c.teamID(teamID)
	}
	return dr.Records, err
}

// SRVUpdate defines the updatable fields within an SRV block of a DNS record.
type SRVUpdate struct {
	Port     *int64  `json:"port"`
	Priority *int64  `json:"priority"`
	Target   *string `json:"target"`
	Weight   *int64  `json:"weight"`
}

// UpdateDNSRecordRequest defines the structure of the request body for updating a DNS record.
type UpdateDNSRecordRequest struct {
	MXPriority *int64     `json:"mxPriority,omitempty"`
	Name       *string    `json:"name,omitempty"`
	SRV        *SRVUpdate `json:"srv,omitempty"`
	TTL        *int64     `json:"ttl,omitempty"`
	Value      *string    `json:"value,omitempty"`
	Comment    string     `json:"comment"`
}

// UpdateDNSRecord updates a DNS record for a specified domain name within Vercel.
func (c *Client) UpdateDNSRecord(ctx context.Context, teamID, recordID string, request UpdateDNSRecordRequest) (r DNSRecord, err error) {
	url := fmt.Sprintf("%s/v4/domains/records/%s", c.baseURL, recordID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating DNS record", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &r)
	r.TeamID = c.teamID(teamID)
	return r, err
}
