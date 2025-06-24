package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// RollingReleaseStage represents a stage in a rolling release
type RollingReleaseStage struct {
	TargetPercentage int  `json:"targetPercentage"`          // Required: 0-100
	Duration         *int `json:"duration,omitempty"`        // Required for automatic advancement: 1-10000 minutes
	RequireApproval  bool `json:"requireApproval,omitempty"` // Only in response for manual-approval type
}

// RollingRelease represents the rolling release configuration
type RollingRelease struct {
	Enabled         bool                  `json:"enabled"`         // Required
	AdvancementType string                `json:"advancementType"` // Required when enabled=true: 'automatic' or 'manual-approval'
	Stages          []RollingReleaseStage `json:"stages"`          // Required when enabled=true: 2-10 stages
}

type RollingReleaseInfo struct {
	RollingRelease RollingRelease `json:"rollingRelease"`
	ProjectID      string         `json:"projectId"`
	TeamID         string         `json:"teamId"`
}

// GetRollingRelease returns the rolling release for a given project.
func (c *Client) GetRollingRelease(ctx context.Context, projectID, teamID string) (RollingReleaseInfo, error) {
	teamId := c.TeamID(teamID)
	url := fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamId=%s", c.baseURL, projectID, teamId)

	var d RollingReleaseInfo
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &d)

	if err != nil {
		return RollingReleaseInfo{}, fmt.Errorf("error getting rolling-release: %w", err)
	}

	d.ProjectID = projectID
	d.TeamID = teamId

	return d, nil
}

// UpdateRollingReleaseRequest defines the information that needs to be passed to Vercel in order to
// update a rolling release.
type UpdateRollingReleaseRequest struct {
	RollingRelease RollingRelease `json:"rollingRelease"`
	ProjectID      string         `json:"projectId"`
	TeamID         string         `json:"teamId"`
}

// UpdateRollingRelease will update an existing rolling release to the latest information.
func (c *Client) UpdateRollingRelease(ctx context.Context, request UpdateRollingReleaseRequest) (RollingReleaseInfo, error) {
	request.TeamID = c.TeamID(request.TeamID)
	if request.RollingRelease.Enabled {
		enableRequest := map[string]any{
			"enabled":         true,
			"advancementType": request.RollingRelease.AdvancementType,
			"stages":          request.RollingRelease.Stages,
		}

		var result RollingReleaseInfo
		err := c.doRequest(clientRequest{
			ctx:    ctx,
			method: "PATCH",
			url:    fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamId=%s", c.baseURL, request.ProjectID, request.TeamID),
			body:   string(mustMarshal(enableRequest)),
		}, &result)
		if err != nil {
			return RollingReleaseInfo{}, fmt.Errorf("error enabling rolling release: %w", err)
		}

		result.ProjectID = request.ProjectID
		result.TeamID = request.TeamID
		tflog.Info(ctx, "enabled rolling release", map[string]any{
			"response": result,
			"request":  request,
		})
		return result, nil
	} else {
		// For disabling, just send the request as is
		disabledRequest := UpdateRollingReleaseRequest{
			RollingRelease: RollingRelease{
				Enabled:         false,
				AdvancementType: "",
				Stages:          []RollingReleaseStage{},
			},
		}

		var result RollingReleaseInfo
		err := c.doRequest(clientRequest{
			ctx:    ctx,
			method: "PATCH",
			url:    fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamId=%s", c.baseURL, request.ProjectID, request.TeamID),
			body:   string(mustMarshal(disabledRequest.RollingRelease)),
		}, &result)
		if err != nil {
			return RollingReleaseInfo{}, fmt.Errorf("error disabling rolling release: %w", err)
		}

		result.ProjectID = request.ProjectID
		result.TeamID = request.TeamID

		return result, nil
	}
}

// DeleteRollingRelease will delete the rolling release for a given project.
func (c *Client) DeleteRollingRelease(ctx context.Context, projectID, teamID string) error {
	teamId := c.TeamID(teamID)
	url := fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamId=%s", c.baseURL, projectID, teamId)

	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
	return err
}
