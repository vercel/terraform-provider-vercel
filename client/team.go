package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// TeamCreateRequest defines the information needed to create a team within vercel.
type TeamCreateRequest struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// Team is the information returned by the vercel api when a team is created.
type Team struct {
	ID                                 string  `json:"id"`
	SensitiveEnvironmentVariablePolicy *string `json:"sensitiveEnvironmentVariablePolicy"`
}

// CreateTeam creates a team within vercel.
func (c *Client) CreateTeam(ctx context.Context, request TeamCreateRequest) (r Team, err error) {
	url := fmt.Sprintf("%s/v1/teams", c.baseURL)

	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating team", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &r)
	return r, err
}

// DeleteTeam deletes an existing team within vercel.
func (c *Client) DeleteTeam(ctx context.Context, teamID string) error {
	url := fmt.Sprintf("%s/v1/teams/%s", c.baseURL, teamID)
	tflog.Info(ctx, "deleting team", map[string]interface{}{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   "",
	}, nil)
}

// GetTeam returns information about an existing team within vercel.
func (c *Client) GetTeam(ctx context.Context, idOrSlug string) (r Team, err error) {
	url := fmt.Sprintf("%s/v2/teams/%s", c.baseURL, idOrSlug)
	tflog.Info(ctx, "getting team", map[string]interface{}{
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
