package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type PasswordProtectionRequest struct {
	DeploymentType string `json:"deploymentType"`
	Password       string `json:"password"`
}

// UpdateProjectRequest defines the possible fields that can be updated within a vercel project.
// note that the values are all pointers, with many containing `omitempty` for serialisation.
// This is because the Vercel API behaves in the following manner:
// - a provided field will be updated
// - setting the field to an empty value (e.g. "") will remove the setting for that field.
// - omitting the value entirely from the request will _not_ update the field.
type UpdateProjectRequest struct {
	BuildCommand                *string                    `json:"buildCommand"`
	CommandForIgnoringBuildStep *string                    `json:"commandForIgnoringBuildStep"`
	DevCommand                  *string                    `json:"devCommand"`
	Framework                   *string                    `json:"framework"`
	InstallCommand              *string                    `json:"installCommand"`
	Name                        *string                    `json:"name,omitempty"`
	OutputDirectory             *string                    `json:"outputDirectory"`
	PublicSource                *bool                      `json:"publicSource"`
	RootDirectory               *string                    `json:"rootDirectory"`
	ServerlessFunctionRegion    *string                    `json:"serverlessFunctionRegion"`
	SSOProtection               *Protection                `json:"ssoProtection"`
	PasswordProtection          *PasswordProtectionRequest `json:"passwordProtection"`
}

// UpdateProject updates an existing projects configuration within Vercel.
func (c *Client) UpdateProject(ctx context.Context, projectID, teamID string, request UpdateProjectRequest, shouldFetchEnvironmentVariables bool) (r ProjectResponse, err error) {
	url := fmt.Sprintf("%s/v8/projects/%s", c.baseURL, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
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
		"url":                             url,
		"payload":                         payload,
		"shouldFetchEnvironmentVariables": shouldFetchEnvironmentVariables,
	})
	err = c.doRequest(req, &r)
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
