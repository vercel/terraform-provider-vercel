package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type Stage struct {
	TargetPercentage float64 `json:"targetPercentage,omitempty"`
	Duration         float64 `json:"duration,omitempty"`
	RequireApproval  bool    `json:"requireApproval,omitempty"`
}

// CreateRollingReleaseRequest defines the information that needs to be passed to Vercel in order to
// create a rolling release.
type RollingReleaseRequest struct {
	Enabled              bool    `json:"enabled,omitempty"`
	AdvancementType      string  `json:"advancementType,omitempty"`
	CanaryResponseHeader string  `json:"canaryResponseHeader,omitempty"`
	Stages               []Stage `json:"stages,omitempty"`
}

// UpdateRollingReleaseRequest defines the information that needs to be passed to Vercel in order to
// update a rolling release.
type UpdateRollingReleaseRequest struct {
	RollingRelease RollingReleaseRequest
	ProjectID      string
	TeamID         string
}

// ProjectRollingRelease defines the rolling release configuration on the Project document.
type ProjectRollingRelease struct {
	Target               string  `json:"target"`
	Stages               []Stage `json:"stages,omitempty"`
	CanaryResponseHeader bool    `json:"canaryResponseHeader,omitempty"`
}

type RollingReleaseResponse struct {
	TeamID string `json:"-"`
}

func (d RollingReleaseResponse) toRollingReleaseResponse(teamID string) RollingReleaseResponse {
	return RollingReleaseResponse{
		TeamID: teamID,
	}
}

// UpdateRollingRelease will update an existing rolling release to the latest information.
func (c *Client) UpdateRollingRelease(ctx context.Context, request UpdateRollingReleaseRequest) (RollingReleaseResponse, error) {
	url := fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamid=%s", c.baseURL, request.ProjectID, request.TeamID)

	payload := string(mustMarshal(request.RollingRelease))

	tflog.Info(ctx, "updating rolling-release", map[string]any{
		"url":     url,
		"payload": payload,
	})
	var d RollingReleaseResponse
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &d)
	return d.toRollingReleaseResponse(c.TeamID(request.TeamID)), err
}

// DeleteRollingRelease will delete the rolling release for a given project.
func (c *Client) DeleteRollingRelease(ctx context.Context, projectID, teamID string) error {
	url := fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamId=%s", c.baseURL, projectID, teamID)

	tflog.Info(ctx, "deleting rolling-release", map[string]any{
		"url": url,
	})

	var d RollingReleaseResponse
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, &d)
	return err
}

// GetRollingRelease returns the rolling release for a given project.
func (c *Client) GetRollingRelease(ctx context.Context, projectID, teamID string) (d RollingReleaseResponse, err error) {
	url := fmt.Sprintf("%s/v1/projects/%s/rolling-release/config?teamId=%s", c.baseURL, projectID, teamID)

	tflog.Info(ctx, "deleting rolling-release", map[string]any{
		"url": url,
	})

	var d RollingReleaseResponse
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &d)
	return err
}
