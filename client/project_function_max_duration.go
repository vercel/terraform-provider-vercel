package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type ProjectFunctionMaxDurationRequest struct {
	ProjectID   string
	TeamID      string
	MaxDuration int64
}

type functionMaxDuration struct {
	DefaultFunctionTimeout *int64 `json:"defaultFunctionTimeout"`
}

type ProjectFunctionMaxDuration struct {
	ProjectID   string
	TeamID      string
	MaxDuration *int64
}

func (c *Client) GetProjectFunctionMaxDuration(ctx context.Context, projectID, teamID string) (p ProjectFunctionMaxDuration, err error) {
	url := fmt.Sprintf("%s/v1/projects/%s/resource-config", c.baseURL, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	tflog.Info(ctx, "get project function max duration", map[string]interface{}{
		"url": url,
	})
	var f functionMaxDuration
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &f)
	if err != nil {
		return p, err
	}
	var maxDuration *int64
	if f.DefaultFunctionTimeout != nil {
		maxDuration = f.DefaultFunctionTimeout
	}
	return ProjectFunctionMaxDuration{
		ProjectID:   projectID,
		TeamID:      teamID,
		MaxDuration: maxDuration,
	}, err
}

func (c *Client) UpdateProjectFunctionMaxDuration(ctx context.Context, request ProjectFunctionMaxDurationRequest) (p ProjectFunctionMaxDuration, err error) {
	url := fmt.Sprintf("%s/v1/projects/%s/resource-config", c.baseURL, request.ProjectID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}

	payload := string(mustMarshal(functionMaxDuration{
		DefaultFunctionTimeout: &request.MaxDuration,
	}))
	var f functionMaxDuration
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &f)
	if err != nil {
		return p, err
	}
	var maxDuration *int64
	if f.DefaultFunctionTimeout != nil {
		maxDuration = f.DefaultFunctionTimeout
	}
	return ProjectFunctionMaxDuration{
		ProjectID:   request.ProjectID,
		TeamID:      request.TeamID,
		MaxDuration: maxDuration,
	}, err
}
