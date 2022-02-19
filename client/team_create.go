package client

import (
	"context"
	"net/http"
	"strings"
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
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/v1/teams",
		strings.NewReader(string(mustMarshal(request))),
	)
	if err != nil {
		return r, err
	}

	err = c.doRequest(req, &r)
	return r, err
}
