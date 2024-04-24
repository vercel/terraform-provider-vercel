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

	tflog.Info(ctx, "creating environment variable", map[string]interface{}{
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
	url := fmt.Sprintf("%s/v10/projects/%s/env", c.baseURL, request.ProjectID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	payload := string(mustMarshal(request.EnvironmentVariables))
	tflog.Info(ctx, "creating environment variables", map[string]interface{}{
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

// UpdateEnvironmentVariableRequest defines the information that needs to be passed to Vercel in order to
// update an environment variable.
type UpdateEnvironmentVariableRequest struct {
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
	tflog.Info(ctx, "updating environment variable", map[string]interface{}{
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

// DeleteEnvironmentVariable will remove an environment variable from Vercel.
func (c *Client) DeleteEnvironmentVariable(ctx context.Context, projectID, teamID, variableID string) error {
	url := fmt.Sprintf("%s/v8/projects/%s/env/%s", c.baseURL, projectID, variableID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	tflog.Info(ctx, "deleting environment variable", map[string]interface{}{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   "",
	}, nil)
}

func (c *Client) GetEnvironmentVariables(ctx context.Context, projectID, teamID string) ([]EnvironmentVariable, error) {
	url := fmt.Sprintf("%s/v8/projects/%s/env?decrypt=true", c.baseURL, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s&teamId=%s", url, c.teamID(teamID))
	}

	envResponse := struct {
		Env []EnvironmentVariable `json:"envs"`
	}{}
	tflog.Info(ctx, "getting environment variables", map[string]interface{}{
		"url": url,
	})
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &envResponse)
	for _, env := range envResponse.Env {
		env.TeamID = c.teamID(teamID)
	}
	return envResponse.Env, err
}

// GetEnvironmentVariable gets a singluar environment variable from Vercel based on its ID.
func (c *Client) GetEnvironmentVariable(ctx context.Context, projectID, teamID, envID string) (e EnvironmentVariable, err error) {
	url := fmt.Sprintf("%s/v1/projects/%s/env/%s", c.baseURL, projectID, envID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	tflog.Info(ctx, "getting environment variable", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &e)
	e.TeamID = c.teamID(teamID)
	return e, err
}
