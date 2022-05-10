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
	url := fmt.Sprintf("%s/now/aliases/%s", c.baseURL, aliasUID)
	req, err := http.NewRequest(
		"DELETE",
		url,
		nil,
	)
	if err != nil {
		return r, err
	}

	// Add query parameters
	q := req.URL.Query()
	if teamID != "" {
		q.Add("teamId", teamID)
	}
	req.URL.RawQuery = q.Encode()

	tflog.Trace(ctx, "deleting alias", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(req, &r)
	if err != nil {
		return r, fmt.Errorf("url: %s, err: %s", url, err)
	}

	return r, nil
}
