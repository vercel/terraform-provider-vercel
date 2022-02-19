package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// DeleteProjectDomain removes any association of a domain name with a Vercel project.
func (c *Client) DeleteProjectDomain(ctx context.Context, projectID, domain, teamID string) error {
	url := fmt.Sprintf("%s/v8/projects/%s/domains/%s", c.baseURL, projectID, domain)
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
