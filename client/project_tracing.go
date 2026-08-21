package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ProjectTracing represents the tracing configuration for a Vercel project.
type ProjectTracing struct {
	TeamID        string
	ProjectID     string
	Enabled       bool                     `json:"enabled"`
	SamplingRules []TraceDrainSamplingRule `json:"-"`
}

type projectTracingResponse struct {
	Enabled  bool             `json:"enabled"`
	Sampling []drainsSampling `json:"sampling"`
}

type updateProjectTracingPayload struct {
	Enabled  bool             `json:"enabled"`
	Sampling []drainsSampling `json:"sampling"`
}

func (r projectTracingResponse) toProjectTracing(teamID, projectID string) ProjectTracing {
	samplingRules := make([]TraceDrainSamplingRule, 0, len(r.Sampling))
	for _, rule := range r.Sampling {
		samplingRules = append(samplingRules, TraceDrainSamplingRule{
			Rate:        rule.Rate,
			Environment: rule.Env,
			RequestPath: rule.RequestPath,
		})
	}

	return ProjectTracing{
		TeamID:        teamID,
		ProjectID:     projectID,
		Enabled:       r.Enabled,
		SamplingRules: samplingRules,
	}
}

func (c *Client) projectTracingURL(projectID, teamID string) string {
	query := url.Values{}
	query.Set("projectId", projectID)
	if resolvedTeamID := c.TeamID(teamID); resolvedTeamID != "" {
		query.Set("teamId", resolvedTeamID)
	}
	return urlWithQuery(fmt.Sprintf("%s/v1/drains/tracing/config", c.baseURL), query)
}

// GetProjectTracing returns the tracing configuration for a project.
func (c *Client) GetProjectTracing(ctx context.Context, projectID, teamID string) (ProjectTracing, error) {
	requestURL := c.projectTracingURL(projectID, teamID)
	tflog.Info(ctx, "reading project tracing configuration", map[string]any{"url": requestURL})

	var response projectTracingResponse
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    requestURL,
	}, &response)
	return response.toProjectTracing(c.TeamID(teamID), projectID), err
}

// UpdateProjectTracing enables or disables tracing and updates its sampling rules.
func (c *Client) UpdateProjectTracing(ctx context.Context, tracing ProjectTracing) (ProjectTracing, error) {
	requestURL := c.projectTracingURL(tracing.ProjectID, tracing.TeamID)
	sampling := make([]drainsSampling, 0, len(tracing.SamplingRules))
	for _, rule := range tracing.SamplingRules {
		sampling = append(sampling, drainsSampling{
			Type:        "head_sampling",
			Rate:        rule.Rate,
			Env:         rule.Environment,
			RequestPath: rule.RequestPath,
		})
	}

	payload := updateProjectTracingPayload{
		Enabled:  tracing.Enabled,
		Sampling: sampling,
	}
	tflog.Info(ctx, "updating project tracing configuration", map[string]any{"url": requestURL})

	var response projectTracingResponse
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PUT",
		url:    requestURL,
		body:   string(mustMarshal(payload)),
	}, &response)
	return response.toProjectTracing(c.TeamID(tracing.TeamID), tracing.ProjectID), err
}
