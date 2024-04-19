package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type SharedEnvironmentVariableResponse struct {
	Key        string   `json:"key"`
	TeamID     string   `json:"ownerId"`
	ID         string   `json:"id,omitempty"`
	Value      string   `json:"value"`
	Type       string   `json:"type"`
	Target     []string `json:"target"`
	ProjectIDs []string `json:"projectId"`
}

type SharedEnvVarRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type SharedEnvironmentVariableRequest struct {
	Type                 string                `json:"type"`
	ProjectIDs           []string              `json:"projectId"`
	Target               []string              `json:"target"`
	EnvironmentVariables []SharedEnvVarRequest `json:"evs"`
}

type CreateSharedEnvironmentVariableRequest struct {
	EnvironmentVariable SharedEnvironmentVariableRequest
	TeamID              string
}

// CreateSharedEnvironmentVariable will create a brand new shared environment variable if one does not exist.
func (c *Client) CreateSharedEnvironmentVariable(ctx context.Context, request CreateSharedEnvironmentVariableRequest) (e SharedEnvironmentVariableResponse, err error) {
	url := fmt.Sprintf("%s/v1/env", c.baseURL)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	payload := string(mustMarshal(request.EnvironmentVariable))
	tflog.Info(ctx, "creating shared environment variable", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	var response struct {
		Created []SharedEnvironmentVariableResponse `json:"created"`
	}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &response)
	if err != nil {
		return e, err
	}
	if len(response.Created) != 1 {
		return e, fmt.Errorf("expected 1 environment variable to be created, got %d", len(response.Created))
	}
	// Override the value, as it returns the encrypted value.
	response.Created[0].Value = request.EnvironmentVariable.EnvironmentVariables[0].Value
	return response.Created[0], err
}

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
	tflog.Info(ctx, "deleting shared environment variable", map[string]interface{}{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   payload,
	}, nil)
}

func (c *Client) GetSharedEnvironmentVariable(ctx context.Context, teamID, envID string) (e SharedEnvironmentVariableResponse, err error) {
	url := fmt.Sprintf("%s/v1/env/%s", c.baseURL, envID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	tflog.Info(ctx, "getting shared environment variable", map[string]interface{}{
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

func (c *Client) ListSharedEnvironmentVariables(ctx context.Context, teamID string) ([]SharedEnvironmentVariableResponse, error) {
	url := fmt.Sprintf("%s/v1/env/all", c.baseURL)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	tflog.Info(ctx, "listing shared environment variables", map[string]interface{}{
		"url": url,
	})
	res := struct {
		Data []SharedEnvironmentVariableResponse `json:"data"`
	}{}
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &res)
	for _, v := range res.Data {
		v.TeamID = c.teamID(teamID)
	}
	return res.Data, err
}

type UpdateSharedEnvironmentVariableRequest struct {
	Value      string   `json:"value"`
	Type       string   `json:"type"`
	ProjectIDs []string `json:"projectId"`
	Target     []string `json:"target"`
	TeamID     string   `json:"-"`
	EnvID      string   `json:"-"`
}

func (c *Client) UpdateSharedEnvironmentVariable(ctx context.Context, request UpdateSharedEnvironmentVariableRequest) (e SharedEnvironmentVariableResponse, err error) {
	url := fmt.Sprintf("%s/v1/env", c.baseURL)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	payload := string(mustMarshal(struct {
		Updates map[string]UpdateSharedEnvironmentVariableRequest `json:"updates"`
	}{
		Updates: map[string]UpdateSharedEnvironmentVariableRequest{
			request.EnvID: request,
		},
	}))

	tflog.Info(ctx, "updating shared environment variable", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	var response struct {
		Updated []SharedEnvironmentVariableResponse `json:"updated"`
	}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &response)
	if err != nil {
		return e, err
	}
	if len(response.Updated) != 1 {
		return e, fmt.Errorf("expected 1 environment variable to be created, got %d", len(response.Updated))
	}
	// Override the value, as it returns the encrypted value.
	response.Updated[0].Value = request.Value
	return response.Updated[0], err
}
