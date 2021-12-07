package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type GitRepository struct {
	Type string `json:"type"`
	Repo string `json:"repo"`
}

type EnvironmentVariable struct {
	Key    string   `json:"key"`
	Value  string   `json:"value"`
	Target []string `json:"target"`
	Type   string   `json:"type"`
	ID     string   `json:"id"`
}

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

	err = c.doRequest(req, &r)
	return r, err
}
