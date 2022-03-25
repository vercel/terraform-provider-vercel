package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// GetDeployment retrieves information from Vercel about an existing Deployment.
func (c *Client) GetDeployment(ctx context.Context, deploymentID, teamID string) (r DeploymentResponse, err error) {
	url := fmt.Sprintf("%s/v13/deployments/%s", c.baseURL, deploymentID)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		url,
		strings.NewReader(""),
	)
	if err != nil {
		return r, err
	}

	tflog.Trace(ctx, "getting deployment", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(req, &r)
	return r, err
}
