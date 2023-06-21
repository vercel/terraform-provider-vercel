package client

import (
	"context"
	"fmt"
)

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
	for _, record := range dr.Records {
		record.TeamID = c.teamID(teamID)
	}
	return dr.Records, err
}
