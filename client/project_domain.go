package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// CreateProjectDomainRequest defines the information necessary to create a project domain.
// A project domain is an association of a specific domain name to a project. These are typically
// used to assign a domain name to any production deployments, but can also be used to configure
// redirects, or to give specific git branches a domain name.
type CreateProjectDomainRequest struct {
	Name                string `json:"name"`
	GitBranch           string `json:"gitBranch,omitempty"`
	CustomEnvironmentID string `json:"customEnvironmentId,omitempty"`
	Redirect            string `json:"redirect,omitempty"`
	RedirectStatusCode  int64  `json:"redirectStatusCode,omitempty"`
}

// CreateProjectDomain creates a project domain within Vercel.
func (c *Client) CreateProjectDomain(ctx context.Context, projectID, teamID string, request CreateProjectDomainRequest) (r ProjectDomainResponse, err error) {
	url := fmt.Sprintf("%s/v10/projects/%s/domains", c.baseURL, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating project domain", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &r)
	r.TeamID = c.teamID(teamID)
	return r, err
}

// DeleteProjectDomain removes any association of a domain name with a Vercel project.
func (c *Client) DeleteProjectDomain(ctx context.Context, projectID, domain, teamID string) error {
	url := fmt.Sprintf("%s/v8/projects/%s/domains/%s", c.baseURL, projectID, domain)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	tflog.Info(ctx, "deleting project domain", map[string]interface{}{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   "",
	}, nil)
}

// ProjectDomainResponse defines the information that Vercel exposes about a domain that is
// associated with a vercel project.
type ProjectDomainResponse struct {
	Name                string  `json:"name"`
	ProjectID           string  `json:"projectId"`
	TeamID              string  `json:"-"`
	Redirect            *string `json:"redirect"`
	RedirectStatusCode  *int64  `json:"redirectStatusCode"`
	GitBranch           *string `json:"gitBranch"`
	CustomEnvironmentID *string `json:"customEnvironmentId"`
}

// GetProjectDomain retrieves information about a project domain from Vercel.
func (c *Client) GetProjectDomain(ctx context.Context, projectID, domain, teamID string) (r ProjectDomainResponse, err error) {
	url := fmt.Sprintf("%s/v8/projects/%s/domains/%s", c.baseURL, projectID, domain)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	tflog.Info(ctx, "getting project domain", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &r)
	r.TeamID = c.teamID(teamID)
	return r, err
}

// UpdateProjectDomainRequest defines the information necessary to update a project domain.
type UpdateProjectDomainRequest struct {
	GitBranch           *string `json:"gitBranch"`
	CustomEnvironmentID *string `json:"customEnvironmentId,omitempty"`
	Redirect            *string `json:"redirect"`
	RedirectStatusCode  *int64  `json:"redirectStatusCode"`
}

// UpdateProjectDomain updates an existing project domain within Vercel.
func (c *Client) UpdateProjectDomain(ctx context.Context, projectID, domain, teamID string, request UpdateProjectDomainRequest) (r ProjectDomainResponse, err error) {
	url := fmt.Sprintf("%s/v8/projects/%s/domains/%s", c.baseURL, projectID, domain)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating project domain", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &r)
	r.TeamID = c.teamID(teamID)
	return r, err
}
