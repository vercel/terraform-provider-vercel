package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// UpsertEnvironmentVariableRequest defines the information that needs to be passed to Vercel in order to
// create or update an environment variable.
type UpsertEnvironmentVariableRequest EnvironmentVariable

// UpsertEnvironmentVariable will either create a brand new environment variable if one does not exist, or will
// update an existing environment variable to the latest information.
func (c *Client) UpsertEnvironmentVariable(ctx context.Context, projectID, teamID string, request UpsertEnvironmentVariableRequest) error {
	url := fmt.Sprintf("%s/v8/projects/%s/env", c.baseURL, projectID)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	}
	payload := string(mustMarshal(request))
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		url,
		strings.NewReader(payload),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	tflog.Trace(ctx, "upserting environment variable", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	return c.doRequest(req, nil)
}
