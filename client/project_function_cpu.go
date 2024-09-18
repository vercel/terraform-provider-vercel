package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type ProjectFunctionCPURequest struct {
	ProjectID string
	TeamID    string
	CPU       string
}

type functionCPU struct {
	DefaultMemoryType *string `json:"defaultMemoryType"`
}

type ProjectFunctionCPU struct {
	ProjectID string
	TeamID    string
	CPU       *string
}

var toCPUNetwork = map[string]string{
	"basic":       "standard_legacy",
	"standard":    "standard",
	"performance": "performance",
}

var fromCPUNetwork = map[string]string{
	"standard_legacy": "basic",
	"standard":        "standard",
	"performance":     "performance",
}

func (c *Client) GetProjectFunctionCPU(ctx context.Context, projectID, teamID string) (p ProjectFunctionCPU, err error) {
	url := fmt.Sprintf("%s/v1/projects/%s", c.baseURL, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	tflog.Info(ctx, "get project function cpu", map[string]interface{}{
		"url": url,
	})
	var f functionCPU
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &f)
	if err != nil {
		return p, err
	}
	var cpu *string
	if f.DefaultMemoryType != nil {
		v := fromCPUNetwork[*f.DefaultMemoryType]
		cpu = &v
	}
	return ProjectFunctionCPU{
		ProjectID: projectID,
		TeamID:    teamID,
		CPU:       cpu,
	}, err
}

func (c *Client) UpdateProjectFunctionCPU(ctx context.Context, request ProjectFunctionCPURequest) (p ProjectFunctionCPU, err error) {
	url := fmt.Sprintf("%s/v1/projects/%s", c.baseURL, request.ProjectID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}

	v := toCPUNetwork[request.CPU]
	payload := string(mustMarshal(functionCPU{
		DefaultMemoryType: &v,
	}))
	var f functionCPU
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &f)
	if err != nil {
		return p, err
	}
	var cpu *string
	if f.DefaultMemoryType != nil {
		v := fromCPUNetwork[*f.DefaultMemoryType]
		cpu = &v
	}
	return ProjectFunctionCPU{
		ProjectID: request.ProjectID,
		TeamID:    request.TeamID,
		CPU:       cpu,
	}, err
}
