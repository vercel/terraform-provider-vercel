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

type OIDCTokenConfig struct {
	Enabled    bool   `json:"enabled"`
	IssuerMode string `json:"issuerMode,omitempty"`
}

// EnvironmentVariable defines the information Vercel requires and surfaces about an environment variable
// that is associated with a project.
type EnvironmentVariable struct {
	Key                  string   `json:"key"`
	Value                string   `json:"value"`
	Target               []string `json:"target"`
	CustomEnvironmentIDs []string `json:"customEnvironmentIds"`
	GitBranch            *string  `json:"gitBranch,omitempty"`
	Type                 string   `json:"type"`
	ID                   string   `json:"id,omitempty"`
	TeamID               string   `json:"-"`
	Comment              string   `json:"comment"`
	Decrypted            *bool    `json:"decrypted"`
}

type DeploymentExpiration struct {
	ExpirationPreview    int `json:"expirationDays"`
	ExpirationProduction int `json:"expirationDaysProduction"`
	ExpirationCanceled   int `json:"expirationDaysCanceled"`
	ExpirationErrored    int `json:"expirationDaysErrored"`
}

// CreateProjectRequest defines the information necessary to create a project.
type CreateProjectRequest struct {
	BuildCommand                      *string               `json:"buildCommand"`
	CommandForIgnoringBuildStep       *string               `json:"commandForIgnoringBuildStep,omitempty"`
	DevCommand                        *string               `json:"devCommand"`
	EnableAffectedProjectsDeployments *bool                 `json:"enableAffectedProjectsDeployments,omitempty"`
	EnvironmentVariables              []EnvironmentVariable `json:"environmentVariables,omitempty"`
	Framework                         *string               `json:"framework"`
	GitRepository                     *GitRepository        `json:"gitRepository,omitempty"`
	InstallCommand                    *string               `json:"installCommand"`
	Name                              string                `json:"name"`
	OIDCTokenConfig                   *OIDCTokenConfig      `json:"oidcTokenConfig,omitempty"`
	OutputDirectory                   *string               `json:"outputDirectory"`
	PublicSource                      *bool                 `json:"publicSource"`
	RootDirectory                     *string               `json:"rootDirectory"`
	ServerlessFunctionRegion          string                `json:"serverlessFunctionRegion,omitempty"`
	ResourceConfig                    *ResourceConfig       `json:"resourceConfig,omitempty"`
	EnablePreviewFeedback             *bool                 `json:"enablePreviewFeedback,omitempty"`
	EnableProductionFeedback          *bool                 `json:"enableProductionFeedback,omitempty"`
}

// CreateProject will create a project within Vercel.
func (c *Client) CreateProject(ctx context.Context, teamID string, request CreateProjectRequest) (r ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v8/projects", c.baseURL)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating project", map[string]any{
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
	tflog.Info(ctx, "deleting project", map[string]any{
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
	DeployHooks      []DeployHook
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
			DeployHooks:      r.Link.DeployHooks,
		}
	case "gitlab":
		return &Repository{
			Type:             "gitlab",
			Repo:             fmt.Sprintf("%s/%s", r.Link.ProjectNamespace, getRepoNameFromURL(r.Link.ProjectURL)),
			ProductionBranch: r.Link.ProductionBranch,
			DeployHooks:      r.Link.DeployHooks,
		}
	case "bitbucket":
		return &Repository{
			Type:             "bitbucket",
			Repo:             fmt.Sprintf("%s/%s", r.Link.Owner, r.Link.Slug),
			ProductionBranch: r.Link.ProductionBranch,
			DeployHooks:      r.Link.DeployHooks,
		}
	}
	return nil
}

// ProjectResponse defines the information Vercel returns about a project.
type ProjectResponse struct {
	BuildCommand                *string `json:"buildCommand"`
	CommandForIgnoringBuildStep *string `json:"commandForIgnoringBuildStep"`
	DevCommand                  *string `json:"devCommand"`
	Framework                   *string `json:"framework"`
	ID                          string  `json:"id"`
	TeamID                      string  `json:"-"`
	InstallCommand              *string `json:"installCommand"`
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
		ProductionBranch *string      `json:"productionBranch"`
		DeployHooks      []DeployHook `json:"deployHooks"`
	} `json:"link"`
	Name                                 string                      `json:"name"`
	OutputDirectory                      *string                     `json:"outputDirectory"`
	PublicSource                         *bool                       `json:"publicSource"`
	RootDirectory                        *string                     `json:"rootDirectory"`
	ServerlessFunctionRegion             *string                     `json:"serverlessFunctionRegion"`
	VercelAuthentication                 *VercelAuthentication       `json:"ssoProtection"`
	PasswordProtection                   *PasswordProtection         `json:"passwordProtection"`
	TrustedIps                           *TrustedIps                 `json:"trustedIps"`
	OIDCTokenConfig                      *OIDCTokenConfig            `json:"oidcTokenConfig"`
	OptionsAllowlist                     *OptionsAllowlist           `json:"optionsAllowlist"`
	ProtectionBypass                     map[string]ProtectionBypass `json:"protectionBypass"`
	AutoExposeSystemEnvVars              *bool                       `json:"autoExposeSystemEnvs"`
	EnablePreviewFeedback                *bool                       `json:"enablePreviewFeedback"`
	EnableProductionFeedback             *bool                       `json:"enableProductionFeedback"`
	EnableAffectedProjectsDeployments    *bool                       `json:"enableAffectedProjectsDeployments"`
	AutoAssignCustomDomains              bool                        `json:"autoAssignCustomDomains"`
	GitLFS                               bool                        `json:"gitLFS"`
	ServerlessFunctionZeroConfigFailover bool                        `json:"serverlessFunctionZeroConfigFailover"`
	CustomerSupportCodeVisibility        bool                        `json:"customerSupportCodeVisibility"`
	GitForkProtection                    bool                        `json:"gitForkProtection"`
	ProductionDeploymentsFastLane        bool                        `json:"productionDeploymentsFastLane"`
	DirectoryListing                     bool                        `json:"directoryListing"`
	SkewProtectionMaxAge                 int                         `json:"skewProtectionMaxAge"`
	GitComments                          *GitComments                `json:"gitComments"`
	Security                             *Security                   `json:"security"`
	DeploymentExpiration                 *DeploymentExpiration       `json:"deploymentExpiration"`
	ResourceConfig                       *ResourceConfigResponse     `json:"resourceConfig"`
	NodeVersion                          string                      `json:"nodeVersion"`
}

type GitComments struct {
	OnCommit      bool `json:"onCommit"`
	OnPullRequest bool `json:"onPullRequest"`
}

type Security struct {
	AttackModeEnabled bool `json:"attackModeEnabled"`
}

type ResourceConfigResponse struct {
	FunctionDefaultMemoryType *string `json:"functionDefaultMemoryType"`
	FunctionDefaultTimeout    *int64  `json:"functionDefaultTimeout"`
	Fluid                     bool    `json:"fluid"`
}

type ResourceConfig struct {
	FunctionDefaultMemoryType *string `json:"functionDefaultMemoryType,omitempty"`
	FunctionDefaultTimeout    *int64  `json:"functionDefaultTimeout,omitempty"`
	Fluid                     *bool   `json:"fluid,omitempty"`
}

// GetProject retrieves information about an existing project from Vercel.
func (c *Client) GetProject(ctx context.Context, projectID, teamID string) (r ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v10/projects/%s", c.baseURL, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	tflog.Info(ctx, "getting project", map[string]any{
		"url": url,
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
	tflog.Info(ctx, "listing projects", map[string]any{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &pr)
	for i := 0; i < len(pr.Projects); i++ {
		pr.Projects[i].TeamID = c.teamID(teamID)
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
	BuildCommand                         *string                         `json:"buildCommand"`
	CommandForIgnoringBuildStep          *string                         `json:"commandForIgnoringBuildStep"`
	DevCommand                           *string                         `json:"devCommand"`
	Framework                            *string                         `json:"framework"`
	InstallCommand                       *string                         `json:"installCommand"`
	Name                                 *string                         `json:"name,omitempty"`
	OutputDirectory                      *string                         `json:"outputDirectory"`
	PublicSource                         *bool                           `json:"publicSource"`
	RootDirectory                        *string                         `json:"rootDirectory"`
	ServerlessFunctionRegion             string                          `json:"serverlessFunctionRegion,omitempty"`
	VercelAuthentication                 *VercelAuthentication           `json:"ssoProtection"`
	PasswordProtection                   *PasswordProtectionWithPassword `json:"passwordProtection"`
	TrustedIps                           *TrustedIps                     `json:"trustedIps"`
	OIDCTokenConfig                      *OIDCTokenConfig                `json:"oidcTokenConfig"`
	OptionsAllowlist                     *OptionsAllowlist               `json:"optionsAllowlist"`
	AutoExposeSystemEnvVars              bool                            `json:"autoExposeSystemEnvs"`
	EnablePreviewFeedback                *bool                           `json:"enablePreviewFeedback"`
	EnableProductionFeedback             *bool                           `json:"enableProductionFeedback"`
	EnableAffectedProjectsDeployments    *bool                           `json:"enableAffectedProjectsDeployments,omitempty"`
	AutoAssignCustomDomains              bool                            `json:"autoAssignCustomDomains"`
	GitLFS                               bool                            `json:"gitLFS"`
	ServerlessFunctionZeroConfigFailover bool                            `json:"serverlessFunctionZeroConfigFailover"`
	CustomerSupportCodeVisibility        bool                            `json:"customerSupportCodeVisibility"`
	GitForkProtection                    bool                            `json:"gitForkProtection"`
	ProductionDeploymentsFastLane        bool                            `json:"productionDeploymentsFastLane"`
	DirectoryListing                     bool                            `json:"directoryListing"`
	SkewProtectionMaxAge                 int                             `json:"skewProtectionMaxAge"`
	GitComments                          *GitComments                    `json:"gitComments"`
	ResourceConfig                       *ResourceConfig                 `json:"resourceConfig,omitempty"`
	NodeVersion                          string                          `json:"nodeVersion,omitempty"`
}

// UpdateProject updates an existing projects configuration within Vercel.
func (c *Client) UpdateProject(ctx context.Context, projectID, teamID string, request UpdateProjectRequest) (r ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v9/projects/%s", c.baseURL, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating project", map[string]any{
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
	tflog.Info(ctx, "updating project production branch", map[string]any{
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
	r.TeamID = c.teamID(c.teamID(request.TeamID))
	return r, err
}

func (c *Client) UnlinkGitRepoFromProject(ctx context.Context, projectID, teamID string) (r ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v9/projects/%s/link", c.baseURL, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	tflog.Info(ctx, "unlinking project git repo", map[string]any{
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
	tflog.Info(ctx, "linking project git repo", map[string]any{
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
	r.TeamID = c.teamID(c.teamID(request.TeamID))
	return r, err
}
