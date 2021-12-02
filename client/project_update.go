package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type UpdateProjectRequest struct {
	Name                 *string               `json:"name"`
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

func (c *Client) UpdateProject(ctx context.Context, projectID, teamID string, request UpdateProjectRequest) (r ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v8/projects/%s", c.baseURL, projectID)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		"PATCH",
		url,
		strings.NewReader(string(mustMarshal(request))),
	)
	if err != nil {
		return r, err
	}

	err = c.doRequest(req, &r)
	return r, err
}
