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
	tflog.Trace(ctx, "creating shared environment variable", map[string]interface{}{
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
