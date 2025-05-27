package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ProjectCrons represents the crons settings for a Vercel project.
type ProjectCrons struct {
	ProjectID string `json:"-"`
	TeamID    string `json:"-"`
	Enabled   bool   `json:"enabled"`
}

// GetProjectCrons retrieves the current crons status for a project.
func (c *Client) GetProjectCrons(ctx context.Context, projectID, teamID string) (ProjectCrons, error) {
	r, err := c.GetProject(ctx, projectID, teamID)

	return ProjectCrons{
		ProjectID: projectID,
		TeamID:    teamID,
		Enabled:   r.Crons == nil || r.Crons.DisabledAt == nil,
	}, err
}

// UpdateProjectCrons toggles the crons feature for a project.
func (c *Client) UpdateProjectCrons(ctx context.Context, request ProjectCrons) (ProjectCrons, error) {
	url := fmt.Sprintf("%s/v1/projects/%s/crons", c.baseURL, request.ProjectID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	tflog.Info(ctx, "updating project crons", map[string]any{
		"url":     url,
		"payload": request,
	})
	var r ProjectResponse
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   string(mustMarshal(request)),
	}, &r)

	return ProjectCrons{
		ProjectID: request.ProjectID,
		TeamID:    request.TeamID,
		Enabled:   r.Crons == nil || r.Crons.DisabledAt == nil,
	}, err
}
