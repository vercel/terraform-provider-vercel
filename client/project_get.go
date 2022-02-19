package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Repository defines the information about a projects git connection.
type Repository struct {
	Type string
	Repo string
}

// Repository is a helper method to convert the ProjectResponse Repository information into a more
// digestible format.
func (r *ProjectResponse) Repository() *Repository {
	if r.Link == nil {
		return nil
	}
	switch r.Link.Type {
	case "github":
		return &Repository{
			Type: "github",
			Repo: fmt.Sprintf("%s/%s", r.Link.Org, r.Link.Repo),
		}
	case "gitlab":
		return &Repository{
			Type: "gitlab",
			Repo: fmt.Sprintf("%s/%s", r.Link.ProjectNamespace, r.Link.ProjectName),
		}
	case "bitbucket":
		return &Repository{
			Type: "bitbucket",
			Repo: fmt.Sprintf("%s/%s", r.Link.Owner, r.Link.Slug),
		}
	}
	return nil
}

// ProjectResponse defines the information vercel returns about a project.
type ProjectResponse struct {
	BuildCommand         *string               `json:"buildCommand"`
	DevCommand           *string               `json:"devCommand"`
	EnvironmentVariables []EnvironmentVariable `json:"env"`
	Framework            *string               `json:"framework"`
	ID                   string                `json:"id"`
	InstallCommand       *string               `json:"installCommand"`
	Link                 *struct {
		Type string `json:"type"`
		// github
		Org  string `json:"org"`
		Repo string `json:"repo"`
		// bitbucket
		Owner string `json:"owner"`
		Slug  string `json:"slug"`
		// gitlab
		ProjectNamespace string `json:"projectNamespace"`
		ProjectName      string `json:"projectName"`
	} `json:"link"`
	Name            string  `json:"name"`
	OutputDirectory *string `json:"outputDirectory"`
	PublicSource    *bool   `json:"publicSource"`
	RootDirectory   *string `json:"rootDirectory"`
}

// GetProject retrieves information about an existing project from vercel.
func (c *Client) GetProject(ctx context.Context, projectID, teamID string) (r ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v8/projects/%s", c.baseURL, projectID)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		url,
		strings.NewReader(""),
	)
	if err != nil {
		return r, err
	}
	err = c.doRequest(req, &r)
	if err != nil {
		return r, err
	}

	env, err := c.getEnvironmentVariables(ctx, projectID, teamID)
	if err != nil {
		return r, fmt.Errorf("error getting environment variables for project: %w", err)
	}
	r.EnvironmentVariables = env
	return r, err
}
