package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// AliasResponse defines the response the Vercel API returns for an alias.
type AliasResponse struct {
	UID   string `json:"uid"`
	Alias string `json:"alias"`
}

// GetAlias retrieves information about an existing alias from vercel.
func (c *Client) GetAlias(ctx context.Context, aliasID, teamID string) (r AliasResponse, err error) {
	url := fmt.Sprintf("%s/now/aliases/%s", c.baseURL, aliasID)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		url,
		nil,
	)
	if err != nil {
		return r, err
	}

	tflog.Trace(ctx, "getting alias", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(req, &r)
	if err != nil {
		return r, err
	}

	return r, nil
}
