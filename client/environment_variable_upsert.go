package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
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
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		url,
		strings.NewReader(string(mustMarshal(request))),
	)
	if err != nil {
		return err
	}

	return c.doRequest(req, nil)
}
