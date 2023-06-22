package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// AliasResponse defines the response the Vercel API returns for an alias.
type AliasResponse struct {
	UID          string `json:"uid"`
	Alias        string `json:"alias"`
	DeploymentID string `json:"deploymentId"`
	TeamID       string `json:"-"`
}

// GetAlias retrieves information about an existing alias from vercel.
func (c *Client) GetAlias(ctx context.Context, alias, teamID string) (r AliasResponse, err error) {
	url := fmt.Sprintf("%s/v4/aliases/%s", c.baseURL, alias)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	tflog.Trace(ctx, "getting alias", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &r)
	r.TeamID = c.teamID(teamID)
	return r, err
}
