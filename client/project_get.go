package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
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

// ProjectResponse defines the information Vercel returns about a project.
type ProjectResponse struct {
	BuildCommand                *string               `json:"buildCommand"`
	CommandForIgnoringBuildStep *string               `json:"commandForIgnoringBuildStep"`
	DevCommand                  *string               `json:"devCommand"`
	EnvironmentVariables        []EnvironmentVariable `json:"env"`
	Framework                   *string               `json:"framework"`
	ID                          string                `json:"id"`
	TeamID                      string                `json:"-"`
	InstallCommand              *string               `json:"installCommand"`
	Link                        *struct {
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
		ProjectID        int64  `json:"projectId,string"`
	} `json:"link"`
	Name                     string  `json:"name"`
	OutputDirectory          *string `json:"outputDirectory"`
	PublicSource             *bool   `json:"publicSource"`
	RootDirectory            *string `json:"rootDirectory"`
	ServerlessFunctionRegion *string `json:"serverlessFunctionRegion"`
}

// GetProject retrieves information about an existing project from Vercel.
func (c *Client) GetProject(ctx context.Context, projectID, teamID string, shouldFetchEnvironmentVariables bool) (r ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v8/projects/%s", c.baseURL, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		url,
		nil,
	)
	if err != nil {
		return r, err
	}
	tflog.Trace(ctx, "getting project", map[string]interface{}{
		"url":                    url,
		"shouldFetchEnvironment": shouldFetchEnvironmentVariables,
	})
	err = c.doRequest(req, &r)
	if err != nil {
		return r, fmt.Errorf("unable to get project: %w", err)
	}

	if shouldFetchEnvironmentVariables {
		r.EnvironmentVariables, err = c.getEnvironmentVariables(ctx, projectID, teamID)
		if err != nil {
			return r, fmt.Errorf("error getting environment variables for project: %w", err)
		}
	} else {
		// The get project endpoint returns environment variables, but returns them fully
		// encrypted. This isn't useful, so we just remove them.
		r.EnvironmentVariables = nil
	}
	r.TeamID = c.teamID(teamID)
	return r, err
}
