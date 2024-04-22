package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// GitRepository is the information Vercel requires and surfaces about which git provider and repository
// a project is linked with.
type GitRepository struct {
	Type string `json:"type"`
	Repo string `json:"repo"`
}

// EnvironmentVariable defines the information Vercel requires and surfaces about an environment variable
// that is associated with a project.
type EnvironmentVariable struct {
	Key       string   `json:"key"`
	Value     string   `json:"value"`
	Target    []string `json:"target"`
	GitBranch *string  `json:"gitBranch,omitempty"`
	Type      string   `json:"type"`
	ID        string   `json:"id,omitempty"`
	TeamID    string   `json:"-"`
}

// CreateProjectRequest defines the information necessary to create a project.
type CreateProjectRequest struct {
	BuildCommand                *string               `json:"buildCommand"`
	CommandForIgnoringBuildStep *string               `json:"commandForIgnoringBuildStep,omitempty"`
	DevCommand                  *string               `json:"devCommand"`
	EnvironmentVariables        []EnvironmentVariable `json:"environmentVariables"`
	Framework                   *string               `json:"framework"`
	GitRepository               *GitRepository        `json:"gitRepository,omitempty"`
	InstallCommand              *string               `json:"installCommand"`
	Name                        string                `json:"name"`
	OutputDirectory             *string               `json:"outputDirectory"`
	PublicSource                *bool                 `json:"publicSource"`
	RootDirectory               *string               `json:"rootDirectory"`
	ServerlessFunctionRegion    *string               `json:"serverlessFunctionRegion,omitempty"`
}

// CreateProject will create a project within Vercel.
func (c *Client) CreateProject(ctx context.Context, teamID string, request CreateProjectRequest) (r ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v8/projects", c.baseURL)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating project", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &r)
	if err != nil {
		return r, err
	}
	env, err := c.getEnvironmentVariables(ctx, r.ID, teamID)
	if err != nil {
		return r, fmt.Errorf("error getting environment variables: %w", err)
	}
	r.EnvironmentVariables = env
	r.TeamID = c.teamID(teamID)
	return r, err
}

// DeleteProject deletes a project within Vercel. Note that there is no need to explicitly
// remove every environment variable, as these cease to exist when a project is removed.
func (c *Client) DeleteProject(ctx context.Context, projectID, teamID string) error {
	url := fmt.Sprintf("%s/v8/projects/%s", c.baseURL, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	tflog.Info(ctx, "deleting project", map[string]interface{}{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   "",
	}, nil)
}

// Repository defines the information about a projects git connection.
type Repository struct {
	Type             string
	Repo             string
	ProductionBranch *string
}

// getRepoNameFromURL is a helper method to extract the repo name from a GitLab URL.
// This is necessary as GitLab doesn't return the repository slug in the API response,
// Because this information isn't present, the only way to obtain it is to parse the URL.
func getRepoNameFromURL(url string) string {
	url = strings.TrimSuffix(url, ".git")
	urlParts := strings.Split(url, "/")

	return urlParts[len(urlParts)-1]
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
			Type:             "github",
			Repo:             fmt.Sprintf("%s/%s", r.Link.Org, r.Link.Repo),
			ProductionBranch: r.Link.ProductionBranch,
		}
	case "gitlab":
		return &Repository{
			Type:             "gitlab",
			Repo:             fmt.Sprintf("%s/%s", r.Link.ProjectNamespace, getRepoNameFromURL(r.Link.ProjectURL)),
			ProductionBranch: r.Link.ProductionBranch,
		}
	case "bitbucket":
		return &Repository{
			Type:             "bitbucket",
			Repo:             fmt.Sprintf("%s/%s", r.Link.Owner, r.Link.Slug),
			ProductionBranch: r.Link.ProductionBranch,
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
		ProjectURL       string `json:"projectUrl"`
		ProjectID        int64  `json:"projectId,string"`
		// production branch
		ProductionBranch *string `json:"productionBranch"`
	} `json:"link"`
	Name                     string                      `json:"name"`
	OutputDirectory          *string                     `json:"outputDirectory"`
	PublicSource             *bool                       `json:"publicSource"`
	RootDirectory            *string                     `json:"rootDirectory"`
	ServerlessFunctionRegion *string                     `json:"serverlessFunctionRegion"`
	VercelAuthentication     *VercelAuthentication       `json:"ssoProtection"`
	PasswordProtection       *PasswordProtection         `json:"passwordProtection"`
	TrustedIps               *TrustedIps                 `json:"trustedIps"`
	ProtectionBypass         map[string]ProtectionBypass `json:"protectionBypass"`
	AutoExposeSystemEnvVars  *bool                       `json:"autoExposeSystemEnvs"`
}

// GetProject retrieves information about an existing project from Vercel.
func (c *Client) GetProject(ctx context.Context, projectID, teamID string, shouldFetchEnvironmentVariables bool) (r ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v10/projects/%s", c.baseURL, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	tflog.Info(ctx, "getting project", map[string]interface{}{
		"url":                    url,
		"shouldFetchEnvironment": shouldFetchEnvironmentVariables,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &r)
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

// ListProjects lists the top 100 projects (no pagination) from within Vercel.
func (c *Client) ListProjects(ctx context.Context, teamID string) (r []ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v8/projects?limit=100", c.baseURL)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s&teamId=%s", url, c.teamID(teamID))
	}

	pr := struct {
		Projects []ProjectResponse `json:"projects"`
	}{}
	tflog.Info(ctx, "listing projects", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &pr)
	for _, p := range pr.Projects {
		p.TeamID = c.teamID(teamID)
	}
	return pr.Projects, err
}

// UpdateProjectRequest defines the possible fields that can be updated within a vercel project.
// note that the values are all pointers, with many containing `omitempty` for serialisation.
// This is because the Vercel API behaves in the following manner:
// - a provided field will be updated
// - setting the field to an empty value (e.g. "") will remove the setting for that field.
// - omitting the value entirely from the request will _not_ update the field.
type UpdateProjectRequest struct {
	BuildCommand                *string                         `json:"buildCommand"`
	CommandForIgnoringBuildStep *string                         `json:"commandForIgnoringBuildStep"`
	DevCommand                  *string                         `json:"devCommand"`
	Framework                   *string                         `json:"framework"`
	InstallCommand              *string                         `json:"installCommand"`
	Name                        *string                         `json:"name,omitempty"`
	OutputDirectory             *string                         `json:"outputDirectory"`
	PublicSource                *bool                           `json:"publicSource"`
	RootDirectory               *string                         `json:"rootDirectory"`
	ServerlessFunctionRegion    *string                         `json:"serverlessFunctionRegion"`
	VercelAuthentication        *VercelAuthentication           `json:"ssoProtection"`
	PasswordProtection          *PasswordProtectionWithPassword `json:"passwordProtection"`
	TrustedIps                  *TrustedIps                     `json:"trustedIps"`
	AutoExposeSystemEnvVars     *bool                           `json:"autoExposeSystemEnvs,omitempty"`
}

// UpdateProject updates an existing projects configuration within Vercel.
func (c *Client) UpdateProject(ctx context.Context, projectID, teamID string, request UpdateProjectRequest, shouldFetchEnvironmentVariables bool) (r ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v9/projects/%s", c.baseURL, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating project", map[string]interface{}{
		"url":                             url,
		"payload":                         payload,
		"shouldFetchEnvironmentVariables": shouldFetchEnvironmentVariables,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &r)
	if err != nil {
		return r, err
	}
	if shouldFetchEnvironmentVariables {
		r.EnvironmentVariables, err = c.getEnvironmentVariables(ctx, r.ID, teamID)
		if err != nil {
			return r, fmt.Errorf("error getting environment variables for project: %w", err)
		}
	} else {
		r.EnvironmentVariables = nil
	}

	r.TeamID = c.teamID(teamID)
	return r, err
}

type UpdateProductionBranchRequest struct {
	TeamID    string `json:"-"`
	ProjectID string `json:"-"`
	Branch    string `json:"branch"`
}

func (c *Client) UpdateProductionBranch(ctx context.Context, request UpdateProductionBranchRequest) (r ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v9/projects/%s/branch", c.baseURL, request.ProjectID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating project production branch", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &r)
	if err != nil {
		return r, err
	}
	env, err := c.getEnvironmentVariables(ctx, r.ID, request.TeamID)
	if err != nil {
		return r, fmt.Errorf("error getting environment variables: %w", err)
	}
	r.EnvironmentVariables = env
	r.TeamID = c.teamID(c.teamID(request.TeamID))
	return r, err
}

func (c *Client) UnlinkGitRepoFromProject(ctx context.Context, projectID, teamID string) (r ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v9/projects/%s/link", c.baseURL, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	tflog.Info(ctx, "unlinking project git repo", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, &r)
	if err != nil {
		return r, fmt.Errorf("error unlinking git repo: %w", err)
	}
	env, err := c.getEnvironmentVariables(ctx, r.ID, teamID)
	if err != nil {
		return r, fmt.Errorf("error getting environment variables: %w", err)
	}
	r.EnvironmentVariables = env
	r.TeamID = c.teamID(teamID)
	return r, err
}

type LinkGitRepoToProjectRequest struct {
	ProjectID string `json:"-"`
	TeamID    string `json:"-"`
	Type      string `json:"type"`
	Repo      string `json:"repo"`
}

func (c *Client) LinkGitRepoToProject(ctx context.Context, request LinkGitRepoToProjectRequest) (r ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v9/projects/%s/link", c.baseURL, request.ProjectID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	tflog.Info(ctx, "linking project git repo", map[string]interface{}{
		"url": url,
	})
	payload := string(mustMarshal(request))
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &r)
	if err != nil {
		return r, fmt.Errorf("error linking git repo: %w", err)
	}
	env, err := c.getEnvironmentVariables(ctx, r.ID, request.TeamID)
	if err != nil {
		return r, fmt.Errorf("error getting environment variables: %w", err)
	}
	r.EnvironmentVariables = env
	r.TeamID = c.teamID(c.teamID(request.TeamID))
	return r, err
}