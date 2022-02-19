package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// GetTeam returns information about an existing team within vercel.
func (c *Client) GetTeam(ctx context.Context, teamID, slug string) (r TeamResponse, err error) {
	url := c.baseURL + "/v1/teams"
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	} else if slug != "" {
		url = fmt.Sprintf("%s?slug=%s", url, slug)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		url,
		strings.NewReader(""),
	)
	if err != nil {
		return r, err
	}

	err = c.doRequest(req, &r)
	return r, err
}
