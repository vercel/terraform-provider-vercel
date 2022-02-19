package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// DeleteTeam deletes an existing team within vercel.
func (c *Client) DeleteTeam(ctx context.Context, teamID string) error {
	req, err := http.NewRequestWithContext(
		ctx,
		"DELETE",
		fmt.Sprintf("%s/v1/teams/%s", c.baseURL, teamID),
		strings.NewReader(""),
	)
	if err != nil {
		return err
	}

	return c.doRequest(req, nil)
}
