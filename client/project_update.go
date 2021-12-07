package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type UpdateProjectRequest struct {
	Name            *string        `json:"name,omitempty"`
	BuildCommand    *string        `json:"buildCommand,omitempty"`
	DevCommand      *string        `json:"devCommand,omitempty"`
	Framework       *string        `json:"framework,omitempty"`
	GitRepository   *GitRepository `json:"gitRepository,omitempty"`
	InstallCommand  *string        `json:"installCommand,omitempty"`
	OutputDirectory *string        `json:"outputDirectory,omitempty"`
	PublicSource    *bool          `json:"publicSource,omitempty"`
	RootDirectory   *string        `json:"rootDirectory,omitempty"`
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
	if err != nil {
		return r, err
	}
	env, err := c.getEnvironmentVariables(ctx, r.ID, teamID)
	if err != nil {
		return r, err
	}
	r.EnvironmentVariables = env
	return r, err
}
