package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type MicrofrontendGroupMembership struct {
	MicrofrontendGroupID            string `json:"microfrontendsGroupId"`
	IsDefaultApp                    bool   `json:"isDefaultApp,omitempty"`
	DefaultRoute                    string `json:"defaultRoute,omitempty"`
	RouteObservabilityToThisProject bool   `json:"routeObservabilityToThisProject,omitempty"`
	ProjectID                       string `json:"projectId"`
	Enabled                         bool   `json:"enabled"`
	TeamID                          string `json:"team_id"`
}

type MicrofrontendGroupMembershipResponseAPI struct {
	GroupIds                        []string `json:"groupIds"`
	Enabled                         bool     `json:"enabled"`
	IsDefaultApp                    bool     `json:"isDefaultApp,omitempty"`
	DefaultRoute                    string   `json:"defaultRoute,omitempty"`
	RouteObservabilityToThisProject bool     `json:"routeObservabilityToThisProject,omitempty"`
	TeamID                          string   `json:"team_id"`
	UpdatedAt                       int      `json:"updatedAt"`
}

type MicrofrontendGroupMembershipsResponseAPI struct {
	ID             string                                  `json:"id"`
	Microfrontends MicrofrontendGroupMembershipResponseAPI `json:"microfrontends"`
}

func (c *Client) GetMicrofrontendGroupMembership(ctx context.Context, TeamID string, GroupID string, ProjectID string) (r MicrofrontendGroupMembership, err error) {
	tflog.Info(ctx, "getting microfrontend group", map[string]any{
		"project_id": ProjectID,
		"group_id":   GroupID,
		"team_id":    c.TeamID(TeamID),
	})
	group, err := c.GetMicrofrontendGroup(ctx, GroupID, c.TeamID(TeamID))
	if err != nil {
		return r, err
	}
	tflog.Info(ctx, "getting microfrontend group membership", map[string]any{
		"project_id": ProjectID,
		"group":      group,
	})
	return group.Projects[ProjectID], nil
}

func (c *Client) AddOrUpdateMicrofrontendGroupMembership(ctx context.Context, request MicrofrontendGroupMembership) (r MicrofrontendGroupMembership, err error) {
	tflog.Info(ctx, "adding / updating microfrontend project to group", map[string]any{
		"is_default_app": request.IsDefaultApp,
		"project_id":     request.ProjectID,
		"group_id":       request.MicrofrontendGroupID,
	})
	p, err := c.PatchMicrofrontendGroupMembership(ctx, MicrofrontendGroupMembership{
		ProjectID:                       request.ProjectID,
		TeamID:                          c.TeamID(request.TeamID),
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
	tflog.Info(ctx, "removing microfrontend project from group", map[string]any{
		"project_id": request.ProjectID,
		"group_id":   request.MicrofrontendGroupID,
		"team_id":    c.TeamID(request.TeamID),
	})
	p, err := c.PatchMicrofrontendGroupMembership(ctx, MicrofrontendGroupMembership{
		ProjectID:            request.ProjectID,
		TeamID:               c.TeamID(request.TeamID),
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
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}

	tflog.Info(ctx, "updating microfrontend group membership", map[string]any{
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
		TeamID:                          c.TeamID(request.TeamID),
		Enabled:                         apiResponse.Microfrontends.Enabled,
		IsDefaultApp:                    apiResponse.Microfrontends.IsDefaultApp,
		DefaultRoute:                    apiResponse.Microfrontends.DefaultRoute,
		RouteObservabilityToThisProject: apiResponse.Microfrontends.RouteObservabilityToThisProject,
	}, nil
}
