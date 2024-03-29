package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// DeleteSharedEnvironmentVariable will remove a shared environment variable from Vercel.
func (c *Client) DeleteSharedEnvironmentVariable(ctx context.Context, teamID, variableID string) error {
	url := fmt.Sprintf("%s/v1/env", c.baseURL)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	payload := string(mustMarshal(struct {
		IDs []string `json:"ids"`
	}{
		IDs: []string{
			variableID,
		},
	}))
	tflog.Trace(ctx, "deleting shared environment variable", map[string]interface{}{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   payload,
	}, nil)
}
