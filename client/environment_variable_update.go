package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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
	TeamID   string   `json:"-"`
	EnvID   string   `json:"-"`
}

// UpdateEnvironmentVariable will update an existing environment variable to the latest information.
func (c *Client) UpdateEnvironmentVariable(ctx context.Context, request UpdateEnvironmentVariableRequest) (e EnvironmentVariable, err error) {
	url := fmt.Sprintf("%s/v9/projects/%s/env/%s", c.baseURL, request.ProjectID, request.EnvID)
	if request.TeamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, request.TeamID)
	}
	payload := string(mustMarshal(request))
	req, err := http.NewRequestWithContext(
		ctx,
		"PATCH",
		url,
		strings.NewReader(payload),
	)
	if err != nil {
		return e, err
	}

	tflog.Trace(ctx, "updating environment variable", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(req, &e)
	// The API response returns an encrypted environment variable, but we want to return the decrypted version.
	e.Value = request.Value
	return e, err
}
