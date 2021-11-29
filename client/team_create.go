package client

import (
	"context"
	"net/http"
	"strings"
)

type TeamCreateRequest struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type TeamCreateResponse struct {
	ID string `json:"id"`
}

func (c *Client) CreateTeam(ctx context.Context, request TeamCreateRequest) (r TeamCreateResponse, err error) {
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
