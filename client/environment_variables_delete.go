package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// DeleteEnvironmentVariable will remove an environment variable from Vercel.
func (c *Client) DeleteEnvironmentVariable(ctx context.Context, projectID, teamID, variableID string) error {
	url := fmt.Sprintf("%s/v8/projects/%s/env/%s", c.baseURL, projectID, variableID)
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

	tflog.Info(ctx, "deleting environment variable", map[string]interface{}{
		"url": url,
	})
	return c.doRequest(req, nil)
}
