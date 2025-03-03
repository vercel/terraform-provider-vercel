package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type MicrofrontendGroupMembership struct {
	MicrofrontendGroupID            string `json:"microfrontendsGroupId"`
	IsDefaultApp                    bool   `json:"isDefaultApp"`
	DefaultRoute                    string `json:"defaultRoute"`
	RouteObservabilityToThisProject bool   `json:"routeObservabilityToThisProject"`
	ProjectID                       string `json:"projectId"`
	Enabled                         bool   `json:"enabled"`
	TeamID                          string `json:"team_id"`
}

type MicrofrontendGroupMembershipResponseAPI struct {
	GroupIds                        []string `json:"groupIds"`
	Enabled                         bool     `json:"enabled"`
	IsDefaultApp                    bool     `json:"isDefaultApp"`
	DefaultRoute                    string   `json:"defaultRoute"`
	RouteObservabilityToThisProject bool     `json:"routeObservabilityToThisProject"`
	TeamID                          string   `json:"team_id"`
	UpdatedAt                       int      `json:"updatedAt"`
}

type MicrofrontendGroupMembershipsResponseAPI struct {
	ID             string                                  `json:"id"`
	Microfrontends MicrofrontendGroupMembershipResponseAPI `json:"microfrontends"`
}

func (c *Client) GetMicrofrontendGroupMembership(ctx context.Context, request MicrofrontendGroupMembership) (r MicrofrontendGroupMembership, err error) {
	tflog.Info(ctx, "getting microfrontend group", map[string]interface{}{
		"project_id": request.ProjectID,
		"group_id":   request.MicrofrontendGroupID,
		"team_id":    c.teamID(request.TeamID),
	})
	group, err := c.GetMicrofrontendGroup(ctx, request.MicrofrontendGroupID, c.teamID(request.TeamID))
	if err != nil {
		return r, err
	}
	tflog.Info(ctx, "getting microfrontend group membership", map[string]interface{}{
		"project_id": request.ProjectID,
		"group":      group,
	})
	return group.Projects[request.ProjectID], nil
}

func (c *Client) AddOrUpdateMicrofrontendGroupMembership(ctx context.Context, request MicrofrontendGroupMembership) (r MicrofrontendGroupMembership, err error) {
	tflog.Info(ctx, "adding / updating microfrontend project to group", map[string]interface{}{
		"is_default_app": request.IsDefaultApp,
		"project_id":     request.ProjectID,
		"group_id":       request.MicrofrontendGroupID,
	})
	p, err := c.PatchMicrofrontendGroupMembership(ctx, MicrofrontendGroupMembership{
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

func (c *Client) RemoveMicrofrontendGroupMembership(ctx context.Context, request MicrofrontendGroupMembership) (r MicrofrontendGroupMembership, err error) {
	tflog.Info(ctx, "removing microfrontend project from group", map[string]interface{}{
		"project_id": request.ProjectID,
		"group_id":   request.MicrofrontendGroupID,
		"team_id":    c.teamID(request.TeamID),
	})
	p, err := c.PatchMicrofrontendGroupMembership(ctx, MicrofrontendGroupMembership{
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

func (c *Client) PatchMicrofrontendGroupMembership(ctx context.Context, request MicrofrontendGroupMembership) (r MicrofrontendGroupMembership, err error) {
	url := fmt.Sprintf("%s/projects/%s/microfrontends", c.baseURL, request.ProjectID)
	payload := string(mustMarshal(MicrofrontendGroupMembership{
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

	tflog.Info(ctx, "updating microfrontend group membership", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	apiResponse := MicrofrontendGroupMembershipsResponseAPI{}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &apiResponse)
	if err != nil {
		return r, err
	}
	return MicrofrontendGroupMembership{
		MicrofrontendGroupID:            request.MicrofrontendGroupID,
		ProjectID:                       request.ProjectID,
		TeamID:                          c.teamID(request.TeamID),
		Enabled:                         apiResponse.Microfrontends.Enabled,
		IsDefaultApp:                    apiResponse.Microfrontends.IsDefaultApp,
		DefaultRoute:                    apiResponse.Microfrontends.DefaultRoute,
		RouteObservabilityToThisProject: apiResponse.Microfrontends.RouteObservabilityToThisProject,
	}, nil
}
