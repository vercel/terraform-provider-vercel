package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type ProjectDeploymentCheckSource struct {
	Kind               string `json:"kind"`
	ExternalCheckName  string `json:"externalCheckName,omitempty"`
	Provider           string `json:"provider,omitempty"`
	WebhookID          string `json:"webhookId,omitempty"`
	ExternalResourceID string `json:"externalResourceId,omitempty"`
}

type ProjectDeploymentCheck struct {
	ID              string                       `json:"id"`
	Name            string                       `json:"name"`
	OwnerID         string                       `json:"ownerId"`
	ProjectID       string                       `json:"projectId"`
	IsRerequestable bool                         `json:"isRerequestable"`
	Requires        string                       `json:"requires"`
	Source          ProjectDeploymentCheckSource `json:"source"`
	Blocks          string                       `json:"blocks"`
	Targets         []string                     `json:"targets"`
	Timeout         int64                        `json:"timeout"`
}

type CreateProjectDeploymentCheckRequest struct {
	ProjectID       string                        `json:"-"`
	TeamID          string                        `json:"-"`
	Name            string                        `json:"name"`
	IsRerequestable *bool                         `json:"isRerequestable,omitempty"`
	Requires        string                        `json:"requires"`
	Source          *ProjectDeploymentCheckSource `json:"source,omitempty"`
	Blocks          *string                       `json:"blocks,omitempty"`
	Targets         *[]string                     `json:"targets,omitempty"`
	Timeout         *int64                        `json:"timeout,omitempty"`
}

type UpdateProjectDeploymentCheckRequest struct {
	ProjectID       string    `json:"-"`
	TeamID          string    `json:"-"`
	ID              string    `json:"-"`
	Name            *string   `json:"name,omitempty"`
	IsRerequestable *bool     `json:"isRerequestable,omitempty"`
	Requires        *string   `json:"requires,omitempty"`
	Blocks          *string   `json:"blocks,omitempty"`
	Targets         *[]string `json:"targets,omitempty"`
	Timeout         *int64    `json:"timeout,omitempty"`
}

func (c *Client) projectDeploymentCheckURL(projectID, checkID, teamID string) string {
	path := fmt.Sprintf("/v2/projects/%s/checks", url.PathEscape(projectID))
	if checkID != "" {
		path += "/" + url.PathEscape(checkID)
	}
	u := c.baseURL + path
	if resolvedTeamID := c.TeamID(teamID); resolvedTeamID != "" {
		u += "?teamId=" + url.QueryEscape(resolvedTeamID)
	}
	return u
}

func (c *Client) CreateProjectDeploymentCheck(ctx context.Context, request CreateProjectDeploymentCheckRequest) (ProjectDeploymentCheck, error) {
	u := c.projectDeploymentCheckURL(request.ProjectID, "", request.TeamID)
	tflog.Info(ctx, "creating project deployment check", map[string]any{"url": u})
	var check ProjectDeploymentCheck
	err := c.doRequest(clientRequest{ctx: ctx, method: "POST", url: u, body: string(mustMarshal(request))}, &check)
	return check, err
}

func (c *Client) GetProjectDeploymentCheck(ctx context.Context, projectID, checkID, teamID string) (ProjectDeploymentCheck, error) {
	u := c.projectDeploymentCheckURL(projectID, checkID, teamID)
	tflog.Info(ctx, "getting project deployment check", map[string]any{"url": u})
	var check ProjectDeploymentCheck
	err := c.doRequest(clientRequest{ctx: ctx, method: "GET", url: u}, &check)
	return check, err
}

func (c *Client) UpdateProjectDeploymentCheck(ctx context.Context, request UpdateProjectDeploymentCheckRequest) (ProjectDeploymentCheck, error) {
	u := c.projectDeploymentCheckURL(request.ProjectID, request.ID, request.TeamID)
	tflog.Info(ctx, "updating project deployment check", map[string]any{"url": u})
	var check ProjectDeploymentCheck
	err := c.doRequest(clientRequest{ctx: ctx, method: "PATCH", url: u, body: string(mustMarshal(request))}, &check)
	return check, err
}

func (c *Client) DeleteProjectDeploymentCheck(ctx context.Context, projectID, checkID, teamID string) error {
	u := c.projectDeploymentCheckURL(projectID, checkID, teamID)
	tflog.Info(ctx, "deleting project deployment check", map[string]any{"url": u})
	return c.doRequest(clientRequest{ctx: ctx, method: "DELETE", url: u}, nil)
}
