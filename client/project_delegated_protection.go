package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// DelegatedProtection represents delegated deployment protection settings.
type DelegatedProtection struct {
	ProjectID      string  `json:"-"`
	TeamID         string  `json:"-"`
	ClientID       string  `json:"clientId"`
	ClientSecret   string  `json:"clientSecret,omitempty"`
	CookieName     *string `json:"cookieName,omitempty"`
	CreatedAt      *int64  `json:"createdAt,omitempty"`
	DeploymentType string  `json:"deploymentType"`
	Issuer         string  `json:"issuer"`
	UpdatedAt      *int64  `json:"updatedAt,omitempty"`
}

// CreateDelegatedProtectionRequest defines the delegated protection creation payload.
type CreateDelegatedProtectionRequest struct {
	ProjectID      string
	TeamID         string
	ClientID       string
	ClientSecret   string
	CookieName     *string
	DeploymentType string
	Issuer         string
}

// UpdateDelegatedProtectionRequest defines delegated protection update fields.
type UpdateDelegatedProtectionRequest struct {
	ProjectID      string
	TeamID         string
	ClientID       *string
	ClientSecret   *string
	CookieName     *string
	DeploymentType *string
	Issuer         *string
}

type delegatedProtectionPayload struct {
	ClientID       string  `json:"clientId,omitempty"`
	ClientSecret   string  `json:"clientSecret,omitempty"`
	CookieName     *string `json:"cookieName,omitempty"`
	DeploymentType string  `json:"deploymentType,omitempty"`
	Issuer         string  `json:"issuer,omitempty"`
}

// CreateDelegatedProtection creates delegated protection for a project.
func (c *Client) CreateDelegatedProtection(ctx context.Context, request CreateDelegatedProtectionRequest) (DelegatedProtection, error) {
	url := fmt.Sprintf("%s/v1/projects/%s/protection/delegated", c.baseURL, request.ProjectID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}

	payload := delegatedProtectionPayload{
		ClientID:       request.ClientID,
		ClientSecret:   request.ClientSecret,
		CookieName:     request.CookieName,
		DeploymentType: request.DeploymentType,
		Issuer:         request.Issuer,
	}

	tflog.Info(ctx, "creating project delegated protection", map[string]any{
		"url":     url,
		"project": request.ProjectID,
	})

	var result DelegatedProtection
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   string(mustMarshal(payload)),
	}, &result)
	if err != nil {
		return DelegatedProtection{}, err
	}

	result.ProjectID = request.ProjectID
	result.TeamID = c.TeamID(request.TeamID)
	return result, nil
}

// UpdateDelegatedProtection updates delegated protection for a project.
func (c *Client) UpdateDelegatedProtection(ctx context.Context, request UpdateDelegatedProtectionRequest) (DelegatedProtection, error) {
	url := fmt.Sprintf("%s/v1/projects/%s/protection/delegated", c.baseURL, request.ProjectID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}

	payload := map[string]any{}
	if request.ClientID != nil {
		payload["clientId"] = *request.ClientID
	}
	if request.ClientSecret != nil {
		payload["clientSecret"] = *request.ClientSecret
	}
	if request.CookieName != nil {
		if *request.CookieName == "" {
			payload["cookieName"] = nil
		} else {
			payload["cookieName"] = *request.CookieName
		}
	}
	if request.DeploymentType != nil {
		payload["deploymentType"] = *request.DeploymentType
	}
	if request.Issuer != nil {
		payload["issuer"] = *request.Issuer
	}

	tflog.Info(ctx, "updating project delegated protection", map[string]any{
		"url":     url,
		"project": request.ProjectID,
	})

	var result DelegatedProtection
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   string(mustMarshal(payload)),
	}, &result)
	if err != nil {
		return DelegatedProtection{}, err
	}

	result.ProjectID = request.ProjectID
	result.TeamID = c.TeamID(request.TeamID)
	return result, nil
}

// DeleteDelegatedProtection disables delegated protection for a project.
func (c *Client) DeleteDelegatedProtection(ctx context.Context, projectID, teamID string) error {
	url := fmt.Sprintf("%s/v1/projects/%s/protection/delegated", c.baseURL, projectID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}

	tflog.Info(ctx, "deleting project delegated protection", map[string]any{
		"url":     url,
		"project": projectID,
	})

	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}

// GetProjectDelegatedProtection reads delegated protection through the project read endpoint.
func (c *Client) GetProjectDelegatedProtection(ctx context.Context, projectID, teamID string) (DelegatedProtection, error) {
	project, err := c.GetProject(ctx, projectID, teamID)
	if err != nil {
		return DelegatedProtection{}, err
	}

	if project.DelegatedProtection == nil {
		return DelegatedProtection{}, APIError{
			Code:       "not_found",
			Message:    "Delegated Protection is not enabled for this project",
			StatusCode: 404,
		}
	}

	result := *project.DelegatedProtection
	result.ProjectID = projectID
	result.TeamID = c.TeamID(teamID)
	return result, nil
}
