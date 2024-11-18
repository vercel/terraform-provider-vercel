package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type AccessGroup struct {
	ID     string `json:"accessGroupId"`
	TeamID string `json:"teamId"`
	Name   string `json:"name"`
}

type GetAccessGroupRequest struct {
	AccessGroupID string
	TeamID        string
}

func (c *Client) GetAccessGroup(ctx context.Context, req GetAccessGroupRequest) (r AccessGroup, err error) {
	url := fmt.Sprintf("%s/v1/access-groups/%s", c.baseURL, req.AccessGroupID)
	if c.teamID(req.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(req.TeamID))
	}
	tflog.Info(ctx, "getting access group", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &r)
	if err != nil {
		return r, fmt.Errorf("unable to get access group: %w", err)
	}

	r.TeamID = c.teamID(req.TeamID)
	return r, err
}

type CreateAccessGroupRequest struct {
	TeamID string
	Name   string
}

func (c *Client) CreateAccessGroup(ctx context.Context, req CreateAccessGroupRequest) (r AccessGroup, err error) {
	url := fmt.Sprintf("%s/v1/access-groups", c.baseURL)
	if c.teamID(req.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(req.TeamID))
	}
	payload := string(mustMarshal(
		struct {
			Name string `json:"name"`
		}{
			Name: req.Name,
		},
	))
	tflog.Info(ctx, "creating access group", map[string]interface{}{
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

type UpdateAccessGroupRequest struct {
	AccessGroupID string
	TeamID        string
	Name          string
}

func (c *Client) UpdateAccessGroup(ctx context.Context, req UpdateAccessGroupRequest) (r AccessGroup, err error) {
	url := fmt.Sprintf("%s/v1/access-groups/%s", c.baseURL, req.AccessGroupID)
	if c.teamID(req.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(req.TeamID))
	}
	payload := string(mustMarshal(
		struct {
			Name string `json:"name"`
		}{
			Name: req.Name,
		},
	))
	tflog.Info(ctx, "updating access group", map[string]interface{}{
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

type DeleteAccessGroupRequest struct {
	AccessGroupID string
	TeamID        string
}

func (c *Client) DeleteAccessGroup(ctx context.Context, req DeleteAccessGroupRequest) error {
	url := fmt.Sprintf("%s/v1/access-groups/%s", c.baseURL, req.AccessGroupID)
	if c.teamID(req.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(req.TeamID))
	}
	tflog.Info(ctx, "deleting access group", map[string]interface{}{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   "",
	}, nil)
}
