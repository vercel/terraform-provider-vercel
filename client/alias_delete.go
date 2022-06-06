package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// DeleteAliasResponse defines the response the Vercel API returns when an alias is deleted.
type DeleteAliasResponse struct {
	Status string `json:"status"`
}

// DeleteAlias deletes an alias within Vercel.
func (c *Client) DeleteAlias(ctx context.Context, aliasUID string, teamID string) (r DeleteAliasResponse, err error) {
	url := fmt.Sprintf("%s/v2/aliases/%s", c.baseURL, aliasUID)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	}
	req, err := http.NewRequest(
		"DELETE",
		url,
		nil,
	)
	if err != nil {
		return r, err
	}

	tflog.Trace(ctx, "deleting alias", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(req, &r)
	return r, err
}
