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

	tflog.Trace(ctx, "deleting deployment", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(req, &r)
	if err != nil {
		return r, err
	}

	return r, nil
}
