package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type MicrofrontendProject struct {
	MicrofrontendGroupID            string `json:"microfrontendsGroupId"`
	IsDefaultApp                    bool   `json:"isDefaultApp"`
	DefaultRoute                    string `json:"defaultRoute"`
	RouteObservabilityToThisProject bool   `json:"routeObservabilityToThisProject"`
	ProjectID                       string `json:"projectId"`
	Enabled                         bool   `json:"enabled"`
	TeamID                          string `json:"team_id"`
}

type MicrofrontendProjectResponseAPI struct {
	GroupIds                        []string `json:"groupIds"`
	Enabled                         bool     `json:"enabled"`
	IsDefaultApp                    bool     `json:"isDefaultApp"`
	DefaultRoute                    string   `json:"defaultRoute"`
	RouteObservabilityToThisProject bool     `json:"routeObservabilityToThisProject"`
	TeamID                          string   `json:"team_id"`
	UpdatedAt                       int      `json:"updatedAt"`
}

type MicrofrontendProjectsResponseAPI struct {
	ID             string                          `json:"id"`
	Microfrontends MicrofrontendProjectResponseAPI `json:"microfrontends"`
}

func (c *Client) AddOrUpdateMicrofrontendProject(ctx context.Context, request MicrofrontendProject) (r MicrofrontendProject, err error) {
	tflog.Info(ctx, "adding / updating microfrontend project to group", map[string]interface{}{
		"project_id": request.ProjectID,
		"group_id":   request.MicrofrontendGroupID,
	})
	p, err := c.PatchMicrofrontendProject(ctx, MicrofrontendProject{
		ProjectID:                       request.ProjectID,
		TeamID:                          c.teamID(request.TeamID),
		Enabled:                         true,
		IsDefaultApp:                    request.IsDefaultApp,
		DefaultRoute:                    request.DefaultRoute,
		RouteObservabilityToThisProject: request.RouteObservabilityToThisProject,
		MicrofrontendGroupID:            request.MicrofrontendGroupID,
	})
	if err != nil {
		return r, err
	}
	return p, nil
}

func (c *Client) RemoveMicrofrontendProject(ctx context.Context, request MicrofrontendProject) (r MicrofrontendProject, err error) {
	tflog.Info(ctx, "removing microfrontend project from group", map[string]interface{}{
		"project_id": request.ProjectID,
		"group_id":   request.MicrofrontendGroupID,
	})
	p, err := c.PatchMicrofrontendProject(ctx, MicrofrontendProject{
		ProjectID:            request.ProjectID,
		TeamID:               c.teamID(request.TeamID),
		Enabled:              false,
		MicrofrontendGroupID: request.MicrofrontendGroupID,
	})
	if err != nil {
		return r, err
	}
	return p, nil
}

func (c *Client) PatchMicrofrontendProject(ctx context.Context, request MicrofrontendProject) (r MicrofrontendProject, err error) {
	url := fmt.Sprintf("%s/projects/%s/microfrontends", c.baseURL, request.ProjectID)
	payload := string(mustMarshal(MicrofrontendProject{
		IsDefaultApp:                    request.IsDefaultApp,
		DefaultRoute:                    request.DefaultRoute,
		RouteObservabilityToThisProject: request.RouteObservabilityToThisProject,
		ProjectID:                       request.ProjectID,
		Enabled:                         request.Enabled,
		MicrofrontendGroupID:            request.MicrofrontendGroupID,
	}))
	if !request.Enabled {
		payload = string(mustMarshal(struct {
			ProjectID string `json:"projectId"`
			Enabled   bool   `json:"enabled"`
		}{
			ProjectID: request.ProjectID,
			Enabled:   request.Enabled,
		}))
	}
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}

	tflog.Info(ctx, "creating microfrontend group", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	apiResponse := MicrofrontendProjectsResponseAPI{}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &apiResponse)
	if err != nil {
		return r, err
	}
	return MicrofrontendProject{
		IsDefaultApp:                    apiResponse.Microfrontends.IsDefaultApp,
		DefaultRoute:                    apiResponse.Microfrontends.DefaultRoute,
		RouteObservabilityToThisProject: apiResponse.Microfrontends.RouteObservabilityToThisProject,
	}, nil
}
