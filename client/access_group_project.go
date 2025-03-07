package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type AccessGroupProject struct {
	TeamID        string `json:"teamId"`
	AccessGroupID string `json:"accessGroupId"`
	ProjectID     string `json:"projectId"`
	Role          string `json:"role"`
}

type CreateAccessGroupProjectRequest struct {
	TeamID        string
	AccessGroupID string
	ProjectID     string
	Role          string
}

func (c *Client) CreateAccessGroupProject(ctx context.Context, req CreateAccessGroupProjectRequest) (r AccessGroupProject, err error) {
	url := fmt.Sprintf("%s/v1/access-groups/%s/projects", c.baseURL, req.AccessGroupID)
	if c.teamID(req.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(req.TeamID))
	}
	payload := string(mustMarshal(
		struct {
			Role      string `json:"role"`
			ProjectID string `json:"projectId"`
		}{
			Role:      req.Role,
			ProjectID: req.ProjectID,
		},
	))
	tflog.Info(ctx, "creating access group project", map[string]any{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &r)
	if err != nil {
		return r, err
	}
	r.TeamID = c.teamID(req.TeamID)
	return r, err
}

type GetAccessGroupProjectRequest struct {
	TeamID        string
	AccessGroupID string
	ProjectID     string
}

func (c *Client) GetAccessGroupProject(ctx context.Context, req GetAccessGroupProjectRequest) (r AccessGroupProject, err error) {
	url := fmt.Sprintf("%s/v1/access-groups/%s/projects/%s", c.baseURL, req.AccessGroupID, req.ProjectID)
	if c.teamID(req.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(req.TeamID))
	}
	tflog.Info(ctx, "getting access group project", map[string]any{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &r)

	if err != nil {
		return r, fmt.Errorf("unable to get access group project: %w", err)
	}

	return r, err
}

type UpdateAccessGroupProjectRequest struct {
	TeamID        string
	AccessGroupID string
	ProjectID     string
	Role          string
}

func (c *Client) UpdateAccessGroupProject(ctx context.Context, req UpdateAccessGroupProjectRequest) (r AccessGroupProject, err error) {
	url := fmt.Sprintf("%s/v1/access-groups/%s/projects/%s", c.baseURL, req.AccessGroupID, req.ProjectID)
	if c.teamID(req.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(req.TeamID))
	}
	payload := string(mustMarshal(
		struct {
			Role string `json:"role"`
		}{
			Role: req.Role,
		},
	))
	tflog.Info(ctx, "updating access group project", map[string]any{
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
	r.TeamID = c.teamID(req.TeamID)
	return r, err
}

type DeleteAccessGroupProjectRequest struct {
	TeamID        string
	AccessGroupID string
	ProjectID     string
}

func (c *Client) DeleteAccessGroupProject(ctx context.Context, req DeleteAccessGroupProjectRequest) error {
	url := fmt.Sprintf("%s/v1/access-groups/%s/projects/%s", c.baseURL, req.AccessGroupID, req.ProjectID)
	if c.teamID(req.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(req.TeamID))
	}
	tflog.Info(ctx, "deleting access group project", map[string]any{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   "",
	}, nil)
}
