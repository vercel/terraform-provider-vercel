package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// DeleteProject deletes a project within Vercel. Note that there is no need to explicitly
// remove every environment variable, as these cease to exist when a project is removed.
func (c *Client) DeleteProject(ctx context.Context, projectID, teamID string) error {
	url := fmt.Sprintf("%s/v8/projects/%s", c.baseURL, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	tflog.Info(ctx, "deleting project", map[string]interface{}{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   "",
	}, nil)
}
