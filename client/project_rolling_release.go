package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type RollingReleaseStage struct {
	TargetPercentage float64 `json:"targetPercentage,omitempty"`
	Duration         float64 `json:"duration,omitempty"`
	RequireApproval  bool    `json:"requireApproval,omitempty"`
}

// CreateRollingReleaseRequest defines the information that needs to be passed to Vercel in order to
// create a rolling release.
type RollingRelease struct {
	Enabled              bool                  `json:"enabled,omitempty"`
	AdvancementType      string                `json:"advancementType,omitempty"`
	CanaryResponseHeader bool                  `json:"canaryResponseHeader,omitempty"`
	Stages               []RollingReleaseStage `json:"stages,omitempty"`
}

type RollingReleaseInfo struct {
	RollingRelease RollingRelease `json:"rollingRelease,omitempty"`
	ProjectID      string         `json:"projectId"`
	TeamID         string         `json:"teamId"`
}

// GetRollingRelease returns the rolling release for a given project.
func (c *Client) GetRollingRelease(ctx context.Context, projectID, teamID string) (RollingReleaseInfo, error) {
	url := fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamId=%s", c.baseURL, projectID, teamID)

	tflog.Info(ctx, "deleting rolling-release", map[string]any{
		"url": url,
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
	RollingRelease RollingRelease `json:"rollingRelease,omitempty"`
	ProjectID      string         `json:"projectId,omitempty"`
	TeamID         string         `json:"teamId,omitempty"`
}

// UpdateRollingRelease will update an existing rolling release to the latest information.
func (c *Client) UpdateRollingRelease(ctx context.Context, request UpdateRollingReleaseRequest) (RollingReleaseInfo, error) {
	url := fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamid=%s", c.baseURL, request.ProjectID, request.TeamID)

	payload := string(mustMarshal(request.RollingRelease))

	tflog.Info(ctx, "updating rolling-release", map[string]any{
		"url":     url,
		"payload": payload,
	})
	var d RollingReleaseInfo
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &d)
	d.ProjectID = request.ProjectID
	d.TeamID = request.TeamID
	return d, err
}

// DeleteRollingRelease will delete the rolling release for a given project.
func (c *Client) DeleteRollingRelease(ctx context.Context, projectID, teamID string) error {
	url := fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamId=%s", c.baseURL, projectID, teamID)

	tflog.Info(ctx, "deleting rolling-release", map[string]any{
		"url": url,
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
