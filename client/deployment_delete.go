package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// DeleteDeploymentResponse defines the response the Vercel API returns when a deployment is deleted.
type DeleteDeploymentResponse struct {
	State string `json:"state"`
	UID   string `json:"uid"`
}

// DeleteDeployment deletes a deployment within Vercel.
func (c *Client) DeleteDeployment(ctx context.Context, deploymentID string, teamID string) (r DeleteDeploymentResponse, err error) {
	url := fmt.Sprintf("%s/v13/deployments/%s", c.baseURL, deploymentID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	tflog.Info(ctx, "deleting deployment", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   "",
	}, &r)
	return r, err
}
