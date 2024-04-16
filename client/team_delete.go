package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// DeleteTeam deletes an existing team within vercel.
func (c *Client) DeleteTeam(ctx context.Context, teamID string) error {
	url := fmt.Sprintf("%s/v1/teams/%s", c.baseURL, teamID)
	tflog.Info(ctx, "deleting team", map[string]interface{}{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   "",
	}, nil)
}
