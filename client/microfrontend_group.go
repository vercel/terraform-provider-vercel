package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type MicrofrontendGroup struct {
	ID         string                                  `json:"id"`
	Name       string                                  `json:"name"`
	Slug       string                                  `json:"slug"`
	TeamID     string                                  `json:"team_id"`
	Projects   map[string]MicrofrontendGroupMembership `json:"projects"`
	DefaultApp MicrofrontendGroupMembership            `json:"defaultApp"`
}

type MicrofrontendGroupsAPIResponse struct {
	Groups []struct {
		Group struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Slug     string `json:"slug"`
			TeamID   string `json:"team_id"`
			Projects map[string]struct {
				IsDefaultApp                    bool   `json:"isDefaultApp"`
				DefaultRoute                    string `json:"defaultRoute"`
				RouteObservabilityToThisProject bool   `json:"routeObservabilityToThisProject"`
				ProjectID                       string `json:"projectId"`
				Enabled                         bool   `json:"enabled"`
			} `json:"projects"`
		} `json:"group"`
		Projects []MicrofrontendGroupMembershipsResponseAPI `json:"projects"`
	} `json:"groups"`
}

func (c *Client) CreateMicrofrontendGroup(ctx context.Context, TeamID string, Name string) (r MicrofrontendGroup, err error) {
	if c.TeamID(TeamID) == "" {
		return r, fmt.Errorf("team_id is required")
	}
	tflog.Info(ctx, "creating microfrontend group", map[string]any{
		"microfrontend_group_name": Name,
		"team_id":                  c.TeamID(TeamID),
	})
	url := fmt.Sprintf("%s/teams/%s/microfrontends", c.baseURL, c.TeamID(TeamID))
	payload := string(mustMarshal(struct {
		NewMicrofrontendsGroupName string `json:"newMicrofrontendsGroupName"`
	}{
		NewMicrofrontendsGroupName: Name,
	}))
	apiResponse := struct {
		NewMicrofrontendGroup MicrofrontendGroup `json:"newMicrofrontendsGroup"`
	}{}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &apiResponse)
	if err != nil {
		return r, err
	}
	return MicrofrontendGroup{
		ID:     apiResponse.NewMicrofrontendGroup.ID,
		Name:   apiResponse.NewMicrofrontendGroup.Name,
		Slug:   apiResponse.NewMicrofrontendGroup.Slug,
		TeamID: c.TeamID(TeamID),
	}, nil
}

func (c *Client) UpdateMicrofrontendGroup(ctx context.Context, request MicrofrontendGroup) (r MicrofrontendGroup, err error) {
	if c.TeamID(request.TeamID) == "" {
		return r, fmt.Errorf("team_id is required")
	}
	url := fmt.Sprintf("%s/teams/%s/microfrontends/%s", c.baseURL, c.TeamID(request.TeamID), request.ID)
	payload := string(mustMarshal(struct {
		Name string `json:"name"`
	}{
		Name: request.Name,
	}))
	tflog.Info(ctx, "updating microfrontend group", map[string]any{
		"url":     url,
		"payload": payload,
	})
	apiResponse := struct {
		UpdatedMicrofrontendsGroup MicrofrontendGroup `json:"updatedMicrofrontendsGroup"`
	}{}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &apiResponse)
	if err != nil {
		return r, err
	}
	return MicrofrontendGroup{
		ID:     apiResponse.UpdatedMicrofrontendsGroup.ID,
		Name:   apiResponse.UpdatedMicrofrontendsGroup.Name,
		Slug:   apiResponse.UpdatedMicrofrontendsGroup.Slug,
		TeamID: c.TeamID(request.TeamID),
	}, nil
}

func (c *Client) DeleteMicrofrontendGroup(ctx context.Context, request MicrofrontendGroup) (r struct{}, err error) {
	if c.TeamID(request.TeamID) == "" {
		return r, fmt.Errorf("team_id is required")
	}
	url := fmt.Sprintf("%s/teams/%s/microfrontends/%s", c.baseURL, c.TeamID(request.TeamID), request.ID)

	tflog.Info(ctx, "deleting microfrontend group", map[string]any{
		"url": url,
	})

	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   "",
	}, &r)
	return r, err
}

func (c *Client) GetMicrofrontendGroup(ctx context.Context, microfrontendGroupID string, teamID string) (r MicrofrontendGroup, err error) {
	if c.TeamID(teamID) == "" {
		return r, fmt.Errorf("team_id is required")
	}
	url := fmt.Sprintf("%s/v1/microfrontends/groups", c.baseURL)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}

	tflog.Info(ctx, "getting microfrontend group", map[string]any{
		"url": url,
	})
	out := MicrofrontendGroupsAPIResponse{}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &out)

	if err != nil {
		return r, err
	}

	tflog.Info(ctx, "getting microfrontend group", map[string]any{
		"out": out,
	})

	for i := range out.Groups {
		if out.Groups[i].Group.ID == microfrontendGroupID {
			projects := map[string]MicrofrontendGroupMembership{}
			defaultApp := MicrofrontendGroupMembership{}
			for _, p := range out.Groups[i].Projects {
				projects[p.ID] = MicrofrontendGroupMembership{
					MicrofrontendGroupID:            microfrontendGroupID,
					ProjectID:                       p.ID,
					TeamID:                          c.TeamID(teamID),
					Enabled:                         p.Microfrontends.Enabled,
					IsDefaultApp:                    p.Microfrontends.IsDefaultApp,
					DefaultRoute:                    p.Microfrontends.DefaultRoute,
					RouteObservabilityToThisProject: p.Microfrontends.RouteObservabilityToThisProject,
				}
				if p.Microfrontends.IsDefaultApp {
					defaultApp = projects[p.ID]
				}
			}
			r := MicrofrontendGroup{
				ID:         out.Groups[i].Group.ID,
				Name:       out.Groups[i].Group.Name,
				Slug:       out.Groups[i].Group.Slug,
				TeamID:     c.TeamID(teamID),
				DefaultApp: defaultApp,
				Projects:   projects,
			}
			tflog.Info(ctx, "returning microfrontend group", map[string]any{
				"r": r,
			})
			return r, nil
		}
	}

	return r, fmt.Errorf("microfrontend group not found")
}
