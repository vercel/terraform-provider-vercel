package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// GetDeployment retrieves information from Vercel about an existing Deployment.
func (c *Client) GetDeployment(ctx context.Context, deploymentID, teamID string) (r DeploymentResponse, err error) {
	url := fmt.Sprintf("%s/v13/deployments/%s", c.baseURL, deploymentID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	tflog.Trace(ctx, "getting deployment", map[string]interface{}{
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
