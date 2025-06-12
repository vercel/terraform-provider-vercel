package client

import (
	"context"
	"fmt"
	"sort"
	"time"
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
	teamId := c.TeamID(request.TeamID)
	// If we're enabling, we need to do it in steps
	if request.RollingRelease.Enabled {
		// First ensure it's disabled
		disabledRequest := UpdateRollingReleaseRequest{
			RollingRelease: RollingRelease{
				Enabled:         false,
				AdvancementType: "",
				Stages:          []RollingReleaseStage{},
			},
		}

		err := c.doRequest(clientRequest{
			ctx:    ctx,
			method: "PATCH",
			url:    fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamId=%s", c.baseURL, request.ProjectID, teamId),
			body:   string(mustMarshal(disabledRequest.RollingRelease)),
		}, nil)
		if err != nil {
			return RollingReleaseInfo{}, fmt.Errorf("error disabling rolling release: %w", err)
		}

		// Wait a bit before proceeding
		time.Sleep(2 * time.Second)

		// Sort stages by target percentage
		sort.Slice(request.RollingRelease.Stages, func(i, j int) bool {
			return request.RollingRelease.Stages[i].TargetPercentage < request.RollingRelease.Stages[j].TargetPercentage
		})

		// First set up the stages
		stagesRequest := map[string]any{
			"enabled":         false,
			"advancementType": request.RollingRelease.AdvancementType,
			"stages":          request.RollingRelease.Stages,
		}

		err = c.doRequest(clientRequest{
			ctx:    ctx,
			method: "PATCH",
			url:    fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamId=%s", c.baseURL, request.ProjectID, teamId),
			body:   string(mustMarshal(stagesRequest)),
		}, nil)
		if err != nil {
			return RollingReleaseInfo{}, fmt.Errorf("error configuring stages: %w", err)
		}

		// Wait a bit before enabling
		time.Sleep(2 * time.Second)

		// Finally enable it
		enableRequest := map[string]any{
			"enabled":         true,
			"advancementType": request.RollingRelease.AdvancementType,
			"stages":          request.RollingRelease.Stages,
		}

		var result RollingReleaseInfo

		err = c.doRequest(clientRequest{
			ctx:    ctx,
			method: "PATCH",
			url:    fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamId=%s", c.baseURL, request.ProjectID, teamId),
			body:   string(mustMarshal(enableRequest)),
		}, &result)
		if err != nil {
			return RollingReleaseInfo{}, fmt.Errorf("error enabling rolling release: %w", err)
		}

		result.ProjectID = request.ProjectID
		result.TeamID = teamId

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
			url:    fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamId=%s", c.baseURL, request.ProjectID, teamId),
			body:   string(mustMarshal(disabledRequest.RollingRelease)),
		}, &result)
		if err != nil {
			return RollingReleaseInfo{}, fmt.Errorf("error disabling rolling release: %w", err)
		}

		result.ProjectID = request.ProjectID
		result.TeamID = teamId

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
