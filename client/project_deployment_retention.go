package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// CreateDeploymentRetentionRequest defines the information that needs to be passed to Vercel in order to
// create an deployment retention.
type DeploymentRetentionRequest struct {
	ExpirationPreview    *string `json:"expiration,omitempty"`
	ExpirationProduction *string `json:"expirationProduction,omitempty"`
	ExpirationCanceled   *string `json:"expirationCanceled,omitempty"`
	ExpirationErrored    *string `json:"expirationErrored,omitempty"`
}

// UpdateDeploymentRetentionRequest defines the information that needs to be passed to Vercel in order to
// update an deployment retention.
type UpdateDeploymentRetentionRequest struct {
	DeploymentRetention DeploymentRetentionRequest
	ProjectID           string
	TeamID              string
}

// DeleteDeploymentRetention will remove any existing deployment retention for a given project.
func (c *Client) DeleteDeploymentRetention(ctx context.Context, projectID, teamID string) error {
	url := fmt.Sprintf("%s/v9/projects/%s/deployment-expiration", c.baseURL, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	unlimited := "unlimited"
	payload := string(mustMarshal(DeploymentRetentionRequest{ExpirationPreview: &unlimited, ExpirationProduction: &unlimited, ExpirationCanceled: &unlimited, ExpirationErrored: &unlimited}))

	tflog.Info(ctx, "updating deployment expiration", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, nil)
	return err
}

type DeploymentExpirationResponse struct {
	DeploymentExpiration DeploymentExpiration `json:"deploymentExpiration"`
}

// UpdateDeploymentRetention will update an existing deployment retention to the latest information.
func (c *Client) UpdateDeploymentRetention(ctx context.Context, request UpdateDeploymentRetentionRequest) (DeploymentExpiration, error) {
	url := fmt.Sprintf("%s/v9/projects/%s/deployment-expiration", c.baseURL, request.ProjectID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	payload := string(mustMarshal(request.DeploymentRetention))

	tflog.Info(ctx, "updating deployment expiration", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	var d DeploymentExpirationResponse
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &d)
	return d.DeploymentExpiration, err
}

// GetDeploymentRetention returns the deployment retention for a given project.
func (c *Client) GetDeploymentRetention(ctx context.Context, projectID, teamID string) (d DeploymentExpiration, err error) {
	url := fmt.Sprintf("%s/v1/projects/%s", c.baseURL, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	tflog.Info(ctx, "getting deployment retention", map[string]interface{}{
		"url": url,
	})
	var p ProjectResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &p)
	if p.DeploymentExpiration == nil {
		return DeploymentExpiration{}, fmt.Errorf("deployment retention not found")
	}
	return *p.DeploymentExpiration, err
}
