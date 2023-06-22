package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type UpdateProductionBranchRequest struct {
	TeamID    string `json:"-"`
	ProjectID string `json:"-"`
	Branch    string `json:"branch"`
}

func (c *Client) UpdateProductionBranch(ctx context.Context, request UpdateProductionBranchRequest) (r ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v9/projects/%s/branch", c.baseURL, request.ProjectID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Trace(ctx, "updating project production branch", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &r)
	if err != nil {
		return r, err
	}
	env, err := c.getEnvironmentVariables(ctx, r.ID, request.TeamID)
	if err != nil {
		return r, fmt.Errorf("error getting environment variables: %w", err)
	}
	r.EnvironmentVariables = env
	r.TeamID = c.teamID(c.teamID(request.TeamID))
	return r, err
}
