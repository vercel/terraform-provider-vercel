package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// AliasResponse defines the response the Vercel API returns for an alias.
type AliasResponse struct {
	UID          string `json:"uid"`
	Alias        string `json:"alias"`
	DeploymentID string `json:"deploymentId"`
}

// GetAlias retrieves information about an existing alias from vercel.
func (c *Client) GetAlias(ctx context.Context, alias, teamID string) (r AliasResponse, err error) {
	url := fmt.Sprintf("%s/v4/aliases/%s", c.baseURL, alias)
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
		return r, fmt.Errorf("creating request: %s", err)
	}
	tflog.Trace(ctx, "getting alias", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(req, &r)
	return r, err
}
