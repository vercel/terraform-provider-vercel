package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// CreateEnvironmentVariableRequest defines the information that needs to be passed to Vercel in order to
// create an environment variable.
type EnvironmentVariableRequest struct {
	Key       string   `json:"key"`
	Value     string   `json:"value"`
	Target    []string `json:"target"`
	GitBranch *string  `json:"gitBranch,omitempty"`
	Type      string   `json:"type"`
}

type CreateEnvironmentVariableRequest struct {
	EnvironmentVariable EnvironmentVariableRequest
	ProjectID           string
	TeamID              string
}

// CreateEnvironmentVariable will create a brand new environment variable if one does not exist.
func (c *Client) CreateEnvironmentVariable(ctx context.Context, request CreateEnvironmentVariableRequest) (e EnvironmentVariable, err error) {
	url := fmt.Sprintf("%s/v9/projects/%s/env", c.baseURL, request.ProjectID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	payload := string(mustMarshal(request.EnvironmentVariable))

	tflog.Trace(ctx, "creating environment variable", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &e)
	// The API response returns an encrypted environment variable, but we want to return the decrypted version.
	e.Value = request.EnvironmentVariable.Value
	e.TeamID = c.teamID(request.TeamID)
	return e, err
}

type CreateEnvironmentVariablesRequest struct {
	EnvironmentVariables []EnvironmentVariableRequest
	ProjectID            string
	TeamID               string
}

func (c *Client) CreateEnvironmentVariables(ctx context.Context, request CreateEnvironmentVariablesRequest) error {
	url := fmt.Sprintf("%s/v9/projects/%s/env", c.baseURL, request.ProjectID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	payload := string(mustMarshal(request.EnvironmentVariables))
	tflog.Trace(ctx, "creating environment variables", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, nil)
}
