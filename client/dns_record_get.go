package client

import (
	"context"
	"fmt"
)

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
