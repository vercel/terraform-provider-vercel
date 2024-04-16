package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// GetTeam returns information about an existing team within vercel.
func (c *Client) GetTeam(ctx context.Context, idOrSlug string) (r TeamResponse, err error) {
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
