package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// CreateDeploymentRetentionRequest defines the information that needs to be passed to Vercel in order to
// create an deployment retention.
type DeploymentRetentionRequest struct {
	ExpirationPreview    string `json:"expiration,omitempty"`
	ExpirationProduction string `json:"expirationProduction,omitempty"`
	ExpirationCanceled   string `json:"expirationCanceled,omitempty"`
	ExpirationErrored    string `json:"expirationErrored,omitempty"`
}

// UpdateDeploymentRetentionRequest defines the information that needs to be passed to Vercel in order to
// update an deployment retention.
type UpdateDeploymentRetentionRequest struct {
	DeploymentRetention DeploymentRetentionRequest
	ProjectID           string
	TeamID              string
}

type DeploymentExpirationResponse struct {
	DeploymentExpiration
	TeamID string `json:"-"`
}

// DeleteDeploymentRetention will remove any existing deployment retention for a given project.
func (c *Client) DeleteDeploymentRetention(ctx context.Context, projectID, teamID string) error {
	url := fmt.Sprintf("%s/v9/projects/%s/deployment-expiration", c.baseURL, projectID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	unlimited := "unlimited"
	payload := string(mustMarshal(DeploymentRetentionRequest{ExpirationPreview: unlimited, ExpirationProduction: unlimited, ExpirationCanceled: unlimited, ExpirationErrored: unlimited}))

	tflog.Info(ctx, "updating deployment expiration", map[string]any{
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

type deploymentExpirationResponse struct {
	DeploymentExpiration struct {
		Expiration           string `json:"expiration"`
		ExpirationProduction string `json:"expirationProduction"`
		ExpirationCanceled   string `json:"expirationCanceled"`
		ExpirationErrored    string `json:"expirationErrored"`
	} `json:"deploymentExpiration"`
}

var DeploymentRetentionDaysToString = map[int]string{
	1:     "1d",
	7:     "1w",
	30:    "1m",
	60:    "2m",
	90:    "3m",
	180:   "6m",
	365:   "1y",
	36500: "unlimited",
}

var DeploymentRetentionStringToDays = map[string]int{
	"1d":        1,
	"1w":        7,
	"1m":        30,
	"2m":        60,
	"3m":        90,
	"6m":        180,
	"1y":        365,
	"unlimited": 36500,
}

func (d deploymentExpirationResponse) toDeploymentExpirationResponse(teamID string) DeploymentExpirationResponse {
	return DeploymentExpirationResponse{
		DeploymentExpiration: DeploymentExpiration{
			ExpirationPreview:    DeploymentRetentionStringToDays[d.DeploymentExpiration.Expiration],
			ExpirationProduction: DeploymentRetentionStringToDays[d.DeploymentExpiration.ExpirationProduction],
			ExpirationCanceled:   DeploymentRetentionStringToDays[d.DeploymentExpiration.ExpirationCanceled],
			ExpirationErrored:    DeploymentRetentionStringToDays[d.DeploymentExpiration.ExpirationErrored],
		},
		TeamID: teamID,
	}
}

// UpdateDeploymentRetention will update an existing deployment retention to the latest information.
func (c *Client) UpdateDeploymentRetention(ctx context.Context, request UpdateDeploymentRetentionRequest) (DeploymentExpirationResponse, error) {
	url := fmt.Sprintf("%s/v9/projects/%s/deployment-expiration", c.baseURL, request.ProjectID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	payload := string(mustMarshal(request.DeploymentRetention))

	tflog.Info(ctx, "updating deployment expiration", map[string]any{
		"url":     url,
		"payload": payload,
	})
	var d deploymentExpirationResponse
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &d)
	return d.toDeploymentExpirationResponse(c.TeamID(request.TeamID)), err
}

// GetDeploymentRetention returns the deployment retention for a given project.
func (c *Client) GetDeploymentRetention(ctx context.Context, projectID, teamID string) (d DeploymentExpirationResponse, err error) {
	url := fmt.Sprintf("%s/v2/projects/%s", c.baseURL, projectID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}

	tflog.Info(ctx, "getting deployment retention", map[string]any{
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
		return DeploymentExpirationResponse{
			DeploymentExpiration: DeploymentExpiration{
				ExpirationPreview:    36500,
				ExpirationProduction: 36500,
				ExpirationCanceled:   36500,
				ExpirationErrored:    36500,
			},
			TeamID: c.TeamID(teamID),
		}, nil
	}
	return DeploymentExpirationResponse{
		DeploymentExpiration: *p.DeploymentExpiration,
		TeamID:               c.TeamID(teamID),
	}, err
}
