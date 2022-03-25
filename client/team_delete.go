package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// DeleteTeam deletes an existing team within vercel.
func (c *Client) DeleteTeam(ctx context.Context, teamID string) error {
	url := fmt.Sprintf("%s/v1/teams/%s", c.baseURL, teamID)
	req, err := http.NewRequestWithContext(
		ctx,
		"DELETE",
		url,
		strings.NewReader(""),
	)
	if err != nil {
		return err
	}

	tflog.Trace(ctx, "deleting team", map[string]interface{}{
		"url": url,
	})
	return c.doRequest(req, nil)
}
