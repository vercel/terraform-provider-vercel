package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type UpdateSharedEnvironmentVariableRequest struct {
	Key        string   `json:"key"`
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
	req, err := http.NewRequestWithContext(
		ctx,
		"PATCH",
		url,
		strings.NewReader(payload),
	)
	if err != nil {
		return e, err
	}

	tflog.Trace(ctx, "updating shared environment variable", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	var response struct {
		Updated []SharedEnvironmentVariableResponse `json:"updated"`
	}
	err = c.doRequest(req, &response)
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
