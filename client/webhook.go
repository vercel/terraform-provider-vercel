package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type CreateWebhookRequest struct {
	TeamID     string   `json:"-"`
	Events     []string `json:"events"`
	Endpoint   string   `json:"url"`
	ProjectIDs []string `json:"projectIds,omitempty"`
}

type Webhook struct {
	Events     []string `json:"events"`
	ID         string   `json:"id"`
	Endpoint   string   `json:"url"`
	TeamID     string   `json:"ownerId"`
	ProjectIDs []string `json:"projectIds"`
	Secret     string   `json:"secret"`
}

func (c *Client) CreateWebhook(ctx context.Context, request CreateWebhookRequest) (w Webhook, err error) {
	url := fmt.Sprintf("%s/v1/webhooks", c.baseURL)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating webhook", map[string]any{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &w)
	return w, err
}

func (c *Client) DeleteWebhook(ctx context.Context, id, teamID string) error {
	url := fmt.Sprintf("%s/v1/webhooks/%s", c.baseURL, id)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "deleting webhook", map[string]any{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}

func (c *Client) GetWebhook(ctx context.Context, id, teamID string) (w Webhook, err error) {
	url := fmt.Sprintf("%s/v1/webhooks/%s", c.baseURL, id)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "getting webhook", map[string]any{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &w)
	return w, err
}
