package client

import (
	"context"
	"fmt"
	"net/http"

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

	tflog.Info(ctx, "deleting deployment", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(req, &r)
	return r, err
}
