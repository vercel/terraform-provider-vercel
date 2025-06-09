package client

import (
	"context"
	"encoding/json"
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

	// Validate last stage is 100%
	lastStage := r.Stages[len(r.Stages)-1]
	if lastStage.TargetPercentage != 100 {
		return fmt.Errorf("last stage must have target_percentage=100, got: %d", lastStage.TargetPercentage)
	}

	// Validate stages are in ascending order and within bounds
	prevPercentage := 0
	for i, stage := range r.Stages {
		// Validate percentage bounds
		if stage.TargetPercentage < 1 || stage.TargetPercentage > 100 {
			return fmt.Errorf("stage %d: target_percentage must be between 1 and 100, got: %d", i, stage.TargetPercentage)
		}

		// Validate ascending order
		if stage.TargetPercentage <= prevPercentage {
			return fmt.Errorf("stage %d: target_percentage must be greater than previous stage (%d), got: %d", i, prevPercentage, stage.TargetPercentage)
		}
		prevPercentage = stage.TargetPercentage

		// Validate duration for automatic advancement
		if r.AdvancementType == "automatic" {
			if stage.Duration == nil || *stage.Duration < 1 || *stage.Duration > 10000 {
				return fmt.Errorf("stage %d: duration must be between 1 and 10000 minutes for automatic advancement, got: %d", i, *stage.Duration)
			}
		}
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

	tflog.Debug(ctx, "getting rolling-release configuration", map[string]any{
		"url":        url,
		"method":     "GET",
		"project_id": projectID,
		"team_id":    teamID,
	})

	d := RollingReleaseInfo{}
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &d)
	d.ProjectID = projectID
	d.TeamID = teamID
	return d, err
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
	// Validate the request
	if err := request.RollingRelease.Validate(); err != nil {
		return RollingReleaseInfo{}, fmt.Errorf("invalid rolling release configuration: %w", err)
	}

	url := fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamId=%s", c.baseURL, request.ProjectID, request.TeamID)

	// Process stages to ensure final stage only has targetPercentage
	stages := make([]map[string]any, len(request.RollingRelease.Stages))
	for i, stage := range request.RollingRelease.Stages {
		if i == len(request.RollingRelease.Stages)-1 {
			// Final stage should only have targetPercentage
			stages[i] = map[string]any{
				"targetPercentage": stage.TargetPercentage,
			}
		} else {
			// Other stages can have all properties
			stageMap := map[string]any{
				"targetPercentage": stage.TargetPercentage,
				"requireApproval":  stage.RequireApproval,
			}
			// Only include duration if it's set
			if stage.Duration != nil {
				stageMap["duration"] = *stage.Duration
			}
			stages[i] = stageMap
		}
	}

	// Send just the rolling release configuration, not the whole request
	payload := string(mustMarshal(map[string]any{
		"enabled":         request.RollingRelease.Enabled,
		"advancementType": request.RollingRelease.AdvancementType,
		"stages":          stages,
	}))

	tflog.Debug(ctx, "updating rolling-release configuration", map[string]any{
		"url":              url,
		"method":           "PATCH",
		"project_id":       request.ProjectID,
		"team_id":          request.TeamID,
		"payload":          payload,
		"base_url":         c.baseURL,
		"enabled":          request.RollingRelease.Enabled,
		"advancement_type": request.RollingRelease.AdvancementType,
		"stages_count":     len(request.RollingRelease.Stages),
	})

	// Log each stage for debugging
	for i, stage := range stages {
		tflog.Debug(ctx, fmt.Sprintf("stage %d configuration", i), map[string]any{
			"stage": stage,
		})
	}

	var d RollingReleaseInfo
	resp, err := c.doRequestWithResponse(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	})

	// Always log the raw response for debugging
	tflog.Debug(ctx, "received raw response", map[string]any{
		"response": resp,
	})

	if err != nil {
		// Try to parse error response
		var errResp ErrorResponse
		if resp != "" && json.Unmarshal([]byte(resp), &errResp) == nil {
			tflog.Error(ctx, "error updating rolling-release", map[string]any{
				"error_code":    errResp.Error.Code,
				"error_message": errResp.Error.Message,
				"url":           url,
				"payload":       payload,
				"response":      resp,
			})
			return d, fmt.Errorf("failed to update rolling release: %s - %s", errResp.Error.Code, errResp.Error.Message)
		}

		tflog.Error(ctx, "error updating rolling-release", map[string]any{
			"error":    err.Error(),
			"url":      url,
			"payload":  payload,
			"response": resp,
		})
		return d, fmt.Errorf("failed to update rolling release: %w", err)
	}

	// Return the request state since we know it's valid
	result := RollingReleaseInfo{
		ProjectID: request.ProjectID,
		TeamID:    request.TeamID,
		RollingRelease: RollingRelease{
			Enabled:         request.RollingRelease.Enabled,
			AdvancementType: request.RollingRelease.AdvancementType,
			Stages:          make([]RollingReleaseStage, len(request.RollingRelease.Stages)),
		},
	}

	// Copy stages, preserving the duration and requireApproval for non-final stages
	for i, stage := range request.RollingRelease.Stages {
		if i == len(request.RollingRelease.Stages)-1 {
			// For the final stage, only include targetPercentage
			result.RollingRelease.Stages[i] = RollingReleaseStage{
				TargetPercentage: stage.TargetPercentage,
				// Do not include Duration or RequireApproval for final stage
			}
		} else {
			// For other stages, include all properties
			result.RollingRelease.Stages[i] = stage
		}
	}

	tflog.Debug(ctx, "returning rolling release configuration", map[string]any{
		"project_id":       result.ProjectID,
		"team_id":          result.TeamID,
		"enabled":          result.RollingRelease.Enabled,
		"advancement_type": result.RollingRelease.AdvancementType,
		"stages":           result.RollingRelease.Stages,
	})

	return result, nil
}

// DeleteRollingRelease will delete the rolling release for a given project.
func (c *Client) DeleteRollingRelease(ctx context.Context, projectID, teamID string) error {
	url := fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamId=%s", c.baseURL, projectID, teamID)

	tflog.Debug(ctx, "deleting rolling-release configuration", map[string]any{
		"url":        url,
		"method":     "DELETE",
		"project_id": projectID,
		"team_id":    teamID,
	})

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
