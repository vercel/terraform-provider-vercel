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

// TeamResponse is the information returned by the vercel api when a team is created.
type TeamResponse struct {
	ID string `json:"id"`
}

// CreateTeam creates a team within vercel.
func (c *Client) CreateTeam(ctx context.Context, request TeamCreateRequest) (r TeamResponse, err error) {
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
