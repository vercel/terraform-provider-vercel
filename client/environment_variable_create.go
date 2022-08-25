package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// CreateEnvironmentVariableRequest defines the information that needs to be passed to Vercel in order to
// create an environment variable.
type CreateEnvironmentVariableRequest struct {
	Key       string   `json:"key"`
	Value     string   `json:"value"`
	Target    []string `json:"target"`
	GitBranch *string  `json:"gitBranch,omitempty"`
	Type      string   `json:"type"`
	ProjectID string   `json:"-"`
	TeamID   string   `json:"-"`
}

// CreateEnvironmentVariable will create a brand new environment variable if one does not exist.
func (c *Client) CreateEnvironmentVariable(ctx context.Context, request CreateEnvironmentVariableRequest) (e EnvironmentVariable, err error) {
	url := fmt.Sprintf("%s/v9/projects/%s/env", c.baseURL, request.ProjectID)
	if request.TeamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, request.TeamID)
	}
	payload := string(mustMarshal(request))
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		url,
		strings.NewReader(payload),
	)
	if err != nil {
		return e, err
	}

	tflog.Trace(ctx, "creating environment variable", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(req, &e)
	// The API response returns an encrypted environment variable, but we want to return the decrypted version.
	e.Value = request.Value
	return e, err
}
