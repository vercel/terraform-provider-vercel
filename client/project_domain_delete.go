package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
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
		nil,
	)
	if err != nil {
		return err
	}

	tflog.Trace(ctx, "deleting project domain", map[string]interface{}{
		"url": url,
	})
	return c.doRequest(req, nil)
}
