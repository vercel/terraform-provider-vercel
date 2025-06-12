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

// ErrorResponse represents the error response from the Vercel API
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Validate checks if the rolling release configuration is valid according to API requirements
func (r *RollingRelease) Validate() error {
	if !r.Enabled {
		return nil // No validation needed when disabled
	}

	// Validate advancement type
	if r.AdvancementType == "" {
		return fmt.Errorf("advancement_type is required when enabled is true")
	}
	if r.AdvancementType != "automatic" && r.AdvancementType != "manual-approval" {
		return fmt.Errorf("advancement_type must be 'automatic' or 'manual-approval' when enabled is true, got: %s", r.AdvancementType)
	}

	// Validate stages
	if len(r.Stages) == 0 {
		return fmt.Errorf("stages are required when enabled is true")
	}
	if len(r.Stages) < 2 || len(r.Stages) > 10 {
		return fmt.Errorf("must have between 2 and 10 stages when enabled is true, got: %d", len(r.Stages))
	}

	// Sort stages by target percentage to ensure correct order
	sort.Slice(r.Stages, func(i, j int) bool {
		return r.Stages[i].TargetPercentage < r.Stages[j].TargetPercentage
	})

	// Validate stages are in ascending order and within bounds
	prevPercentage := 0
	for i, stage := range r.Stages {
		// Validate percentage bounds
		if stage.TargetPercentage < 0 || stage.TargetPercentage > 100 {
			return fmt.Errorf("stage %d: target_percentage must be between 0 and 100, got: %d", i, stage.TargetPercentage)
		}

		// Validate ascending order
		if stage.TargetPercentage <= prevPercentage {
			return fmt.Errorf("stage %d: target_percentage must be greater than previous stage (%d), got: %d", i, prevPercentage, stage.TargetPercentage)
		}
		prevPercentage = stage.TargetPercentage

		// Validate duration for automatic advancement
		if r.AdvancementType == "automatic" {
			if i < len(r.Stages)-1 { // All stages except last need duration
				if stage.Duration == nil {
					return fmt.Errorf("stage %d: duration is required for automatic advancement (except for the last stage)", i)
				}
				if *stage.Duration < 1 || *stage.Duration > 10000 {
					return fmt.Errorf("stage %d: duration must be between 1 and 10000 minutes for automatic advancement, got: %d", i, *stage.Duration)
				}
			} else { // Last stage should not have duration
				if stage.Duration != nil {
					return fmt.Errorf("stage %d: last stage should not have duration for automatic advancement", i)
				}
			}
		} else {
			// For manual approval, no stages should have duration
			if stage.Duration != nil {
				return fmt.Errorf("stage %d: duration should not be set for manual-approval advancement type", i)
			}
		}
	}

	// Validate last stage is 100%
	lastStage := r.Stages[len(r.Stages)-1]
	if lastStage.TargetPercentage != 100 {
		return fmt.Errorf("last stage must have target_percentage=100, got: %d", lastStage.TargetPercentage)
	}

	return nil
}

type RollingReleaseInfo struct {
	RollingRelease RollingRelease `json:"rollingRelease"`
	ProjectID      string         `json:"projectId"`
	TeamID         string         `json:"teamId"`
}

// GetRollingRelease returns the rolling release for a given project.
func (c *Client) GetRollingRelease(ctx context.Context, projectID, teamID string) (RollingReleaseInfo, error) {
	url := fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamId=%s", c.baseURL, projectID, teamID)

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
	d.TeamID = teamID

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
	// If we're enabling, we need to do it in steps
	if request.RollingRelease.Enabled {
		// First ensure it's disabled
		disabledRequest := UpdateRollingReleaseRequest{
			RollingRelease: RollingRelease{
				Enabled:         false,
				AdvancementType: "",
				Stages:          []RollingReleaseStage{},
			},
			ProjectID: request.ProjectID,
			TeamID:    request.TeamID,
		}

		err := c.doRequest(clientRequest{
			ctx:    ctx,
			method: "PATCH",
			url:    fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamId=%s", c.baseURL, request.ProjectID, request.TeamID),
			body:   string(mustMarshal(disabledRequest.RollingRelease)),
		}, nil)
		if err != nil {
			return RollingReleaseInfo{}, fmt.Errorf("error disabling rolling release: %w", err)
		}

		// Wait a bit before proceeding
		time.Sleep(2 * time.Second)

		// Now validate the request
		if err := request.RollingRelease.Validate(); err != nil {
			return RollingReleaseInfo{}, fmt.Errorf("invalid rolling release configuration: %w", err)
		}

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
			url:    fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamId=%s", c.baseURL, request.ProjectID, request.TeamID),
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
			url:    fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamId=%s", c.baseURL, request.ProjectID, request.TeamID),
			body:   string(mustMarshal(enableRequest)),
		}, &result)
		if err != nil {
			return RollingReleaseInfo{}, fmt.Errorf("error enabling rolling release: %w", err)
		}

		result.ProjectID = request.ProjectID
		result.TeamID = request.TeamID

		return result, nil
	} else {
		// For disabling, just send the request as is
		disabledRequest := UpdateRollingReleaseRequest{
			RollingRelease: RollingRelease{
				Enabled:         false,
				AdvancementType: "",
				Stages:          []RollingReleaseStage{},
			},
			ProjectID: request.ProjectID,
			TeamID:    request.TeamID,
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
	url := fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamId=%s", c.baseURL, projectID, teamID)

	var d RollingReleaseInfo
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, &d)
	d.ProjectID = projectID
	d.TeamID = teamID
	return err
}
