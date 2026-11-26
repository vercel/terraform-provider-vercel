package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ProjectSpeedInsights represents whether Speed Insights is enabled for a Vercel project.
type ProjectSpeedInsights struct {
	TeamID    string
	ProjectID string
	Enabled   bool
}

type speedInsightsConfiguration struct {
	EnabledAt  *int64 `json:"enabledAt"`
	DisabledAt *int64 `json:"disabledAt"`
}

func (c *speedInsightsConfiguration) enabled() bool {
	if c == nil || c.EnabledAt == nil {
		return false
	}
	var disabledAt int64
	if c.DisabledAt != nil {
		disabledAt = *c.DisabledAt
	}
	return *c.EnabledAt > disabledAt
}

type projectSpeedInsightsResponse struct {
	SpeedInsights *speedInsightsConfiguration `json:"speedInsights"`
	// Analytics is the legacy Speed Insights configuration. The API treats a
	// project with either configuration enabled as having Speed Insights enabled.
	Analytics *speedInsightsConfiguration `json:"analytics"`
}

type toggleSpeedInsightsPayload struct {
	Value bool `json:"value"`
}

type toggleSpeedInsightsResponse struct {
	Value bool `json:"value"`
}

// GetProjectSpeedInsights returns whether Speed Insights is enabled for a project.
func (c *Client) GetProjectSpeedInsights(ctx context.Context, projectID, teamID string) (ProjectSpeedInsights, error) {
	query := url.Values{}
	if resolvedTeamID := c.TeamID(teamID); resolvedTeamID != "" {
		query.Set("teamId", resolvedTeamID)
	}
	requestURL := urlWithQuery(fmt.Sprintf("%s/v9/projects/%s", c.baseURL, url.PathEscape(projectID)), query)
	tflog.Info(ctx, "reading project speed insights", map[string]any{"url": requestURL})

	var response projectSpeedInsightsResponse
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    requestURL,
	}, &response)
	if err != nil {
		return ProjectSpeedInsights{}, err
	}
	return ProjectSpeedInsights{
		TeamID:    c.TeamID(teamID),
		ProjectID: projectID,
		Enabled:   response.SpeedInsights.enabled() || response.Analytics.enabled(),
	}, nil
}

// UpdateProjectSpeedInsights enables or disables Speed Insights for a project.
func (c *Client) UpdateProjectSpeedInsights(ctx context.Context, speedInsights ProjectSpeedInsights) (ProjectSpeedInsights, error) {
	query := url.Values{}
	query.Set("projectId", speedInsights.ProjectID)
	if resolvedTeamID := c.TeamID(speedInsights.TeamID); resolvedTeamID != "" {
		query.Set("teamId", resolvedTeamID)
	}
	requestURL := urlWithQuery(fmt.Sprintf("%s/speed-insights/toggle", c.baseURL), query)
	tflog.Info(ctx, "updating project speed insights", map[string]any{"url": requestURL, "enabled": speedInsights.Enabled})

	var response toggleSpeedInsightsResponse
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    requestURL,
		body:   string(mustMarshal(toggleSpeedInsightsPayload{Value: speedInsights.Enabled})),
	}, &response)
	if err != nil {
		return ProjectSpeedInsights{}, err
	}
	return ProjectSpeedInsights{
		TeamID:    c.TeamID(speedInsights.TeamID),
		ProjectID: speedInsights.ProjectID,
		Enabled:   response.Value,
	}, nil
}
