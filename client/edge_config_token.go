package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type EdgeConfigToken struct {
	TeamID       string
	Token        string `json:"token"`
	Label        string `json:"label"`
	ID           string `json:"id"`
	EdgeConfigID string `json:"edgeConfigId"`
}

func (e EdgeConfigToken) ConnectionString() string {
	return fmt.Sprintf(
		"https://edge-config.vercel.com/%s?token=%s",
		e.EdgeConfigID,
		e.Token,
	)
}

type CreateEdgeConfigTokenRequest struct {
	Label        string `json:"label"`
	TeamID       string `json:"-"`
	EdgeConfigID string `json:"-"`
}

func (c *Client) CreateEdgeConfigToken(ctx context.Context, request CreateEdgeConfigTokenRequest) (e EdgeConfigToken, err error) {
	url := fmt.Sprintf("%s/v1/edge-config/%s/token", c.baseURL, request.EdgeConfigID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating edge config token", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &e)
	e.Label = request.Label
	e.TeamID = request.TeamID
	e.EdgeConfigID = request.EdgeConfigID
	return e, err
}

type EdgeConfigTokenRequest struct {
	TeamID       string
	EdgeConfigID string
	Token        string
}

func (c *Client) DeleteEdgeConfigToken(ctx context.Context, request EdgeConfigTokenRequest) error {
	url := fmt.Sprintf("%s/v1/edge-config/%s/tokens", c.baseURL, request.EdgeConfigID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	payload := string(mustMarshal(
		struct {
			Tokens []string `json:"tokens"`
		}{
			Tokens: []string{request.Token},
		},
	))

	tflog.Info(ctx, "deleting edge config token", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   payload,
	}, nil)
}

func (c *Client) GetEdgeConfigToken(ctx context.Context, request EdgeConfigTokenRequest) (e EdgeConfigToken, err error) {
	url := fmt.Sprintf("%s/v1/edge-config/%s/token/%s", c.baseURL, request.EdgeConfigID, request.Token)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}

	tflog.Info(ctx, "getting edge config token", map[string]interface{}{
		"url": url,
	})

	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &e)
	e.TeamID = request.TeamID
	return e, err
}
