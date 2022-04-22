package client

import (
	"context"
	"fmt"
	"net/http"
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
}

// CreateProjectRequest defines the information necessary to create a project.
type CreateProjectRequest struct {
	Name                 string                `json:"name"`
	BuildCommand         *string               `json:"buildCommand"`
	DevCommand           *string               `json:"devCommand"`
	EnvironmentVariables []EnvironmentVariable `json:"environmentVariables"`
	Framework            *string               `json:"framework"`
	GitRepository        *GitRepository        `json:"gitRepository,omitempty"`
	InstallCommand       *string               `json:"installCommand"`
	OutputDirectory      *string               `json:"outputDirectory"`
	PublicSource         *bool                 `json:"publicSource"`
	RootDirectory        *string               `json:"rootDirectory"`
}

// CreateProject will create a project within Vercel.
func (c *Client) CreateProject(ctx context.Context, teamID string, request CreateProjectRequest) (r ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v8/projects", c.baseURL)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		url,
		strings.NewReader(string(mustMarshal(request))),
	)
	if err != nil {
		return r, err
	}

	tflog.Trace(ctx, "creating project", map[string]interface{}{
		"url":     url,
		"payload": string(mustMarshal(request)),
	})
	err = c.doRequest(req, &r)
	if err != nil {
		return r, err
	}
	env, err := c.getEnvironmentVariables(ctx, r.ID, teamID)
	if err != nil {
		return r, fmt.Errorf("error getting environment variables: %w", err)
	}
	r.EnvironmentVariables = env
	return r, err
}
