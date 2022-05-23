package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// DeleteDNSRecord removes a DNS domain from Vercel.
func (c *Client) DeleteDNSRecord(ctx context.Context, domain, recordID, teamID string) error {
	url := fmt.Sprintf("%s/v2/domains/%s/records/%s", c.baseURL, domain, recordID)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"DELETE",
		url,
		strings.NewReader(""),
	)
	if err != nil {
		return err
	}

	return c.doRequest(req, nil)
}
