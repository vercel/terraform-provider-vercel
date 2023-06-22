package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// UpdateEnvironmentVariableRequest defines the information that needs to be passed to Vercel in order to
// update an environment variable.
type UpdateEnvironmentVariableRequest struct {
	Key       string   `json:"key"`
	Value     string   `json:"value"`
	Target    []string `json:"target"`
	GitBranch *string  `json:"gitBranch,omitempty"`
	Type      string   `json:"type"`
	ProjectID string   `json:"-"`
	TeamID    string   `json:"-"`
	EnvID     string   `json:"-"`
}

// UpdateEnvironmentVariable will update an existing environment variable to the latest information.
func (c *Client) UpdateEnvironmentVariable(ctx context.Context, request UpdateEnvironmentVariableRequest) (e EnvironmentVariable, err error) {
	url := fmt.Sprintf("%s/v9/projects/%s/env/%s", c.baseURL, request.ProjectID, request.EnvID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Trace(ctx, "updating environment variable", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &e)
	// The API response returns an encrypted environment variable, but we want to return the decrypted version.
	e.Value = request.Value
	e.TeamID = c.teamID(request.TeamID)
	return e, err
}
