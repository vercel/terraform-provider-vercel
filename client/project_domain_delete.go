package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// DeleteProjectDomain removes any association of a domain name with a Vercel project.
func (c *Client) DeleteProjectDomain(ctx context.Context, projectID, domain, teamID string) error {
	url := fmt.Sprintf("%s/v8/projects/%s/domains/%s", c.baseURL, projectID, domain)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	tflog.Trace(ctx, "deleting project domain", map[string]interface{}{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   "",
	}, nil)
}
