package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

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

	err = c.doRequest(req, &r)
	return r, err
}
