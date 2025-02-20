package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type MicrofrontendGroup struct {
	ID       string                          `json:"id"`
	Name     string                          `json:"name"`
	Slug     string                          `json:"slug"`
	TeamID   string                          `json:"team_id"`
	Projects map[string]MicrofrontendProject `json:"projects"`
}

type MicrofrontendGroupsAPI struct {
	Group    MicrofrontendGroup                 `json:"group"`
	Projects []MicrofrontendProjectsResponseAPI `json:"projects"`
}

type MicrofrontendGroupsAPIResponse struct {
	Groups []MicrofrontendGroupsAPI `json:"groups"`
}

func (c *Client) CreateMicrofrontendGroup(ctx context.Context, request MicrofrontendGroup) (r MicrofrontendGroup, err error) {
	if c.teamID(request.TeamID) == "" {
		return r, fmt.Errorf("team_id is required")
	}
	tflog.Info(ctx, "creating microfrontend group", map[string]interface{}{
		"microfrontend_group_name": request.Name,
		"team_id":                  c.teamID(request.TeamID),
	})
	url := fmt.Sprintf("%s/teams/%s/microfrontends", c.baseURL, c.teamID(request.TeamID))
	payload := string(mustMarshal(struct {
		NewMicrofrontendsGroupName string `json:"newMicrofrontendsGroupName"`
	}{
		NewMicrofrontendsGroupName: request.Name,
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
		TeamID: c.teamID(request.TeamID),
	}, nil
}

func (c *Client) UpdateMicrofrontendGroup(ctx context.Context, request MicrofrontendGroup) (r MicrofrontendGroup, err error) {
	if c.teamID(request.TeamID) == "" {
		return r, fmt.Errorf("team_id is required")
	}
	url := fmt.Sprintf("%s/teams/%s/microfrontends/%s", c.baseURL, c.teamID(request.TeamID), request.ID)
	payload := string(mustMarshal(struct {
		Name string `json:"name"`
	}{
		Name: request.Name,
	}))
	tflog.Info(ctx, "updating microfrontend group", map[string]interface{}{
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
		TeamID: c.teamID(request.TeamID),
	}, nil
}

func (c *Client) DeleteMicrofrontendGroup(ctx context.Context, request MicrofrontendGroup) (r struct{}, err error) {
	if c.teamID(request.TeamID) == "" {
		return r, fmt.Errorf("team_id is required")
	}
	url := fmt.Sprintf("%s/teams/%s/microfrontends/%s", c.baseURL, c.teamID(request.TeamID), request.ID)

	tflog.Info(ctx, "deleting microfrontend group", map[string]interface{}{
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
	if c.teamID(teamID) == "" {
		return r, fmt.Errorf("team_id is required")
	}
	url := fmt.Sprintf("%s/v1/microfrontends/groups", c.baseURL)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	tflog.Info(ctx, "getting microfrontend group", map[string]interface{}{
		"url": url,
	})
	res := MicrofrontendGroupsAPIResponse{}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &res)

	if err != nil {
		return r, err
	}

	tflog.Info(ctx, "getting microfrontend group", map[string]interface{}{
		"res": res,
	})

	for i := range res.Groups {
		if res.Groups[i].Group.ID == microfrontendGroupID {
			projects := map[string]MicrofrontendProject{}
			for _, p := range res.Groups[i].Projects {
				projects[p.ID] = MicrofrontendProject{
					IsDefaultApp:                    p.Microfrontends.IsDefaultApp,
					DefaultRoute:                    p.Microfrontends.DefaultRoute,
					RouteObservabilityToThisProject: p.Microfrontends.RouteObservabilityToThisProject,
				}
			}
			return MicrofrontendGroup{
				ID:       res.Groups[i].Group.ID,
				Name:     res.Groups[i].Group.Name,
				Slug:     res.Groups[i].Group.Slug,
				TeamID:   teamID,
				Projects: projects,
			}, nil
		}
	}

	return r, fmt.Errorf("microfrontend group not found")
}
