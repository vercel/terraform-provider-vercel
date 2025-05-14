package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type EdgeConfigSchema struct {
	ID         string `json:"-"`
	Definition any    `json:"definition"`
	TeamID     string `json:"-"`
}

func (c *Client) UpsertEdgeConfigSchema(ctx context.Context, request EdgeConfigSchema) (e EdgeConfigSchema, err error) {
	url := fmt.Sprintf("%s/v1/edge-config/%s/schema", c.baseURL, request.ID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating edge config schema", map[string]any{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &e)
	e.TeamID = c.TeamID(request.TeamID)
	e.ID = request.ID
	return e, err
}

func (c *Client) GetEdgeConfigSchema(ctx context.Context, id, teamID string) (e EdgeConfigSchema, err error) {
	url := fmt.Sprintf("%s/v1/edge-config/%s/schema", c.baseURL, id)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "getting edge config schema", map[string]any{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:              ctx,
		method:           "GET",
		url:              url,
		errorOnNoContent: true,
	}, &e)

	if noContent(err) {
		return e, APIError{
			StatusCode: 404,
			Message:    "Edge Config Schema not found",
			Code:       "not_found",
		}
	}

	e.TeamID = c.TeamID(teamID)
	e.ID = id
	return e, err
}

func (c *Client) DeleteEdgeConfigSchema(ctx context.Context, id, teamID string) error {
	url := fmt.Sprintf("%s/v1/edge-config/%s/schema", c.baseURL, id)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "deleting edge config schema", map[string]any{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}
