package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// UpdateProjectRequest defines the possible fields that can be updated within a vercel project.
// note that the values are all pointers, with many containing `omitempty` for serialisation.
// This is because the Vercel API behaves in the following manner:
// - a provided field will be updated
// - setting the field to an empty value (e.g. '') will remove the setting for that field.
// - omitting the value entirely from the request will _not_ update the field.
type UpdateProjectRequest struct {
	BuildCommand                *string `json:"buildCommand"`
	CommandForIgnoringBuildStep *string `json:"commandForIgnoringBuildStep"`
	DevCommand                  *string `json:"devCommand"`
	Framework                   *string `json:"framework"`
	InstallCommand              *string `json:"installCommand"`
	Name                        *string `json:"name,omitempty"`
	OutputDirectory             *string `json:"outputDirectory"`
	PublicSource                *bool   `json:"publicSource"`
	RootDirectory               *string `json:"rootDirectory"`
	ServerlessFunctionRegion    *string `json:"serverlessFunctionRegion"`
}

// UpdateProject updates an existing projects configuration within vercel.
func (c *Client) UpdateProject(ctx context.Context, projectID, teamID string, request UpdateProjectRequest) (r ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v8/projects/%s", c.baseURL, projectID)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	}
	payload := string(mustMarshal(request))
	req, err := http.NewRequestWithContext(
		ctx,
		"PATCH",
		url,
		strings.NewReader(payload),
	)
	if err != nil {
		return r, err
	}

	tflog.Trace(ctx, "updating project", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(req, &r)
	if err != nil {
		return r, err
	}
	env, err := c.getEnvironmentVariables(ctx, r.ID, teamID)
	if err != nil {
		return r, fmt.Errorf("error getting environment variables for project: %w", err)
	}
	r.EnvironmentVariables = env
	return r, err
}
