package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type CreateMicrofrontendGroupRequestAPI struct {
	NewMicrofrontendsGroupName string `json:"newMicrofrontendsGroupName"`
}

// CreateMicrofrontendGroupRequest defines the request the Vercel API expects in order to create a microfrontend group.
type CreateMicrofrontendGroupRequest struct {
	Name   string `json:"name"`
	TeamID string `json:"team_id"`
}

// MicrofrontendGroupResponse defines the response the Vercel API returns when a microfrontend group is created or updated.
type NewMicrofrontendGroupResponseAPI struct {
	NewMicrofrontendGroup MicrofrontendGroupResponse `json:"newMicrofrontendsGroup"`
}

type MicrofrontendGroupResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	TeamID string `json:"team_id"`
}

// CreateMicrofrontendGroup creates a microfrontend group within Vercel.
func (c *Client) CreateMicrofrontendGroup(ctx context.Context, request CreateMicrofrontendGroupRequest) (r MicrofrontendGroupResponse, err error) {
	url := fmt.Sprintf("%s/teams/%s/microfrontends", c.baseURL, c.teamID(request.TeamID))
	payload := string(mustMarshal(CreateMicrofrontendGroupRequestAPI{
		NewMicrofrontendsGroupName: request.Name,
	}))

	tflog.Info(ctx, "creating microfrontend group", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	apiResponse := NewMicrofrontendGroupResponseAPI{}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &apiResponse)
	if err != nil {
		return r, err
	}
	return MicrofrontendGroupResponse{
		ID:     apiResponse.NewMicrofrontendGroup.ID,
		Name:   apiResponse.NewMicrofrontendGroup.Name,
		Slug:   apiResponse.NewMicrofrontendGroup.Slug,
		TeamID: c.teamID(request.TeamID),
	}, nil
}

type UpdateMicrofrontendGroupRequestAPI struct {
	Name string `json:"name"`
}
type UpdateMicrofrontendGroupRequest struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	TeamID string `json:"team_id"`
}

type UpdateMicrofrontendGroupResponseAPI struct {
	UpdatedMicrofrontendsGroup MicrofrontendGroupResponseInner `json:"updatedMicrofrontendsGroup"`
}

// UpdateMicrofrontendGroup updates a microfrontend group within Vercel.
func (c *Client) UpdateMicrofrontendGroup(ctx context.Context, request UpdateMicrofrontendGroupRequest) (r MicrofrontendGroupResponse, err error) {
	url := fmt.Sprintf("%s/teams/%s/microfrontends/%s", c.baseURL, c.teamID(request.TeamID), request.ID)
	payload := string(mustMarshal(UpdateMicrofrontendGroupRequestAPI{
		Name: request.Name,
	}))

	tflog.Info(ctx, "updating microfrontend group", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	apiResponse := UpdateMicrofrontendGroupResponseAPI{}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &apiResponse)
	if err != nil {
		return r, err
	}
	return MicrofrontendGroupResponse{
		ID:     apiResponse.UpdatedMicrofrontendsGroup.ID,
		Name:   apiResponse.UpdatedMicrofrontendsGroup.Name,
		Slug:   apiResponse.UpdatedMicrofrontendsGroup.Slug,
		TeamID: c.teamID(request.TeamID),
	}, nil
}

// MicrofrontendGroupResponse defines the response the Vercel API returns when a microfrontend group is deleted.
type DeleteMicrofrontendGroupResponse struct{}

// DeleteMicrofrontendGroup deletes a microfrontend group within Vercel.
func (c *Client) DeleteMicrofrontendGroup(ctx context.Context, microfrontendGroupID string, teamID string) (r DeleteMicrofrontendGroupResponse, err error) {
	url := fmt.Sprintf("%s/teams/%s/microfrontends/%s", c.baseURL, c.teamID(teamID), microfrontendGroupID)

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

type ProjectMicrofrontend struct {
	Enabled      bool     `json:"enabled"`
	GroupIds     []string `json:"groupIds"`
	IsDefaultApp bool     `json:"isDefaultApp"`
	UpdatedAt    string   `json:"updatedAt"`
}

type MicrofrontendProjectResponse struct {
	ID             string               `json:"id"`
	Name           string               `json:"name"`
	Framework      string               `json:"framework"`
	Microfrontends ProjectMicrofrontend `json:"microfrontends"`
}

type MicrofrontendGroupResponseInner struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type MicrofrontendGroupsResponseInner struct {
	Group    MicrofrontendGroupResponseInner `json:"group"`
	Projects []ProjectResponse               `json:"projects"`
}

type MicrofrontendGroupsResponse struct {
	Groups []MicrofrontendGroupsResponseInner `json:"groups"`
}

// GetMicrofrontendGroups retrieves information from Vercel about existing Microfrontend Groups.
func (c *Client) GetMicrofrontendGroups(ctx context.Context, teamID string) (r MicrofrontendGroupsResponse, err error) {
	url := fmt.Sprintf("%s/v1/microfrontends/groups", c.baseURL)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	tflog.Info(ctx, "getting microfrontend group", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &r)
	return r, err
}

// GetMicrofrontendGroup retrieves information from Vercel about an existing MicrofrontendGroup.
func (c *Client) GetMicrofrontendGroup(ctx context.Context, microfrontendGroupID string, teamID string) (r MicrofrontendGroupResponse, err error) {
	res, err := c.GetMicrofrontendGroups(ctx, teamID)

	if err != nil {
		return r, err
	}

	fmt.Print(res)

	for i := range res.Groups {
		if res.Groups[i].Group.ID == microfrontendGroupID {
			return MicrofrontendGroupResponse{
				ID:     res.Groups[i].Group.ID,
				Name:   res.Groups[i].Group.Name,
				Slug:   res.Groups[i].Group.Slug,
				TeamID: teamID,
			}, nil
		}
	}

	return r, fmt.Errorf("microfrontend group not found")
}
