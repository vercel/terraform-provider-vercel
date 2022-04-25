package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		url,
		strings.NewReader(payload),
	)
	if err != nil {
		return r, err
	}

	req.Header.Set("Content-Type", "application/json")

	tflog.Trace(ctx, "creating team", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(req, &r)
	return r, err
}
