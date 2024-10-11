package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type EdgeConfigOperation struct {
	Operation string `json:"operation"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

type EdgeConfigItem struct {
	TeamID       string
	Key          string `json:"key"`
	Value        string `json:"value"`
	EdgeConfigID string `json:"edgeConfigId"`
}

type CreateEdgeConfigItemRequest struct {
	EdgeConfigID string
	TeamID       string
	Key          string
	Value        string
}

func (c *Client) CreateEdgeConfigItem(ctx context.Context, request CreateEdgeConfigItemRequest) (e EdgeConfigItem, err error) {
	url := fmt.Sprintf("%s/v1/edge-config/%s/items", c.baseURL, request.EdgeConfigID)
	teamID := c.teamID(request.TeamID)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	}

	payload := string(mustMarshal(
		struct {
			Items []EdgeConfigOperation `json:"items"`
		}{
			Items: []EdgeConfigOperation{
				{
					Operation: "upsert",
					Key:       request.Key,
					Value:     request.Value,
				},
			},
		},
	))
	tflog.Info(ctx, "creating edge config token", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, nil)

	return EdgeConfigItem{
		Key:          request.Key,
		Value:        request.Value,
		EdgeConfigID: request.EdgeConfigID,
		TeamID:       teamID,
	}, err
}

type EdgeConfigItemRequest struct {
	EdgeConfigID string
	TeamID       string
	Key          string
	Value        string
}

func (c *Client) DeleteEdgeConfigItem(ctx context.Context, request EdgeConfigItemRequest) error {
	url := fmt.Sprintf("%s/v1/edge-config/%s/items", c.baseURL, request.EdgeConfigID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}

	payload := string(mustMarshal(
		struct {
			Items []EdgeConfigOperation `json:"items"`
		}{
			Items: []EdgeConfigOperation{
				{
					Operation: "delete",
					Key:       request.Key,
					Value:     request.Value,
				},
			},
		},
	))

	tflog.Info(ctx, "deleting edge config token", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, nil)
}

func (c *Client) GetEdgeConfigItem(ctx context.Context, request EdgeConfigItemRequest) (e EdgeConfigItem, err error) {
	url := fmt.Sprintf("%s/v1/edge-config/%s/item/%s", c.baseURL, request.EdgeConfigID, request.Key)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}

	tflog.Info(ctx, "getting edge config token", map[string]interface{}{
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
			Message:    "Edge Config Item not found",
			Code:       "not_found",
		}
	}

	e.TeamID = c.teamID(request.TeamID)
	return e, err
}
