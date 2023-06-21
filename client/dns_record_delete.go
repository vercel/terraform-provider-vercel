package client

import (
	"context"
	"fmt"
)

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
