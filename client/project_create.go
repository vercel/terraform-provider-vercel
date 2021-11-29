package client

import (
	"context"
	"net/http"
	"strings"
)

type GitRepository struct {
	Type string `json:"type"`
	Repo string `json:"repo"`
}

type CreateProjectRequest struct {
	Name                 string            `json:"name"`
	BuildCommand         *string           `json:"buildCommand"`
	DevCommand           *string           `json:"devCommand"`
	EnvironmentVariables map[string]string `json:"environmentVariables"`
	Framework            *string           `json:"framework"`
	GitRepository        GitRepository     `json:"gitRepository,omitempty"`
	InstallCommand       *string           `json:"installCommand"`
	OutputDirectory      *string           `json:"outputDirectory"`
	PublicSource         *string           `json:"publicSource"`
	RootDirectory        *string           `json:"rootDirectory"`
}

func (c *Client) CreateProject(ctx context.Context, request CreateProjectRequest, teamID string) (r ProjectResponse, err error) {
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/v8/projects",
		strings.NewReader(string(mustMarshal(request))),
	)
	if err != nil {
		return r, err
	}

	err = c.doRequest(req, &r)
	return r, err
}
