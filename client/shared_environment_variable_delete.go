package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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
	req, err := http.NewRequestWithContext(
		ctx,
		"DELETE",
		url,
		strings.NewReader(payload),
	)
	if err != nil {
		return err
	}

	tflog.Trace(ctx, "deleting shared environment variable", map[string]interface{}{
		"url": url,
	})
	return c.doRequest(req, nil)
}
