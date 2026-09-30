package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	EnvironmentLifecycleSystem = "system"
	EnvironmentLifecycleCustom = "custom"

	EnvironmentManagedByVercel = "vercel"
	EnvironmentManagedByUser   = "user"

	EnvironmentIDProduction  = "production"
	EnvironmentIDPreview     = "preview"
	EnvironmentIDDevelopment = "development"
)

// Environment is the unified representation of a built-in system environment
// (production, preview, development) or a user-managed custom environment (env_*).
type Environment struct {
	ID            string
	Slug          string
	Name          string
	Type          string
	Description   string
	Lifecycle     string
	ManagedBy     string
	BranchMatcher *BranchMatcher
	CreatedAt     *int64
	UpdatedAt     *int64
	Capabilities  EnvironmentCapabilities
	TeamID        string
	ProjectID     string
}

type EnvironmentCapabilities struct {
	Manage      bool
	Domains     bool
	Deployments bool
}

type ListEnvironmentsRequest struct {
	TeamID    string
	ProjectID string
}

type GetEnvironmentRequest struct {
	TeamID    string
	ProjectID string
	// IDOrSlug accepts a stable environment ID (production, preview, development, or env_*),
	// a slug, or a system environment name (Production, Preview, Development).
	IDOrSlug string
}

type listCustomEnvironmentsAPIResponse struct {
	Environments []customEnvironmentJSON `json:"environments"`
}

type customEnvironmentJSON struct {
	ID            string         `json:"id"`
	Slug          string         `json:"slug"`
	Type          string         `json:"type"`
	Description   string         `json:"description"`
	BranchMatcher *BranchMatcher `json:"branchMatcher"`
	CreatedAt     *int64         `json:"createdAt"`
	UpdatedAt     *int64         `json:"updatedAt"`
}

func (c *Client) ListEnvironments(ctx context.Context, request ListEnvironmentsRequest) ([]Environment, error) {
	teamID := c.TeamID(request.TeamID)
	url := fmt.Sprintf("%s/v1/projects/%s/custom-environments", c.baseURL, request.ProjectID)
	if teamID != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, teamID)
	}
	tflog.Info(ctx, "listing environments", map[string]any{
		"url": url,
	})

	var raw listCustomEnvironmentsAPIResponse
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &raw)
	if err != nil {
		return nil, err
	}

	envs := systemEnvironments(teamID, request.ProjectID)
	for _, custom := range raw.Environments {
		envs = append(envs, custom.toEnvironment(teamID, request.ProjectID))
	}
	return envs, nil
}

func (c *Client) GetEnvironment(ctx context.Context, request GetEnvironmentRequest) (Environment, error) {
	envs, err := c.ListEnvironments(ctx, ListEnvironmentsRequest{
		TeamID:    request.TeamID,
		ProjectID: request.ProjectID,
	})
	if err != nil {
		return Environment{}, err
	}

	for _, env := range envs {
		if environmentMatches(env, request.IDOrSlug) {
			return env, nil
		}
	}

	return Environment{}, APIError{
		StatusCode: 404,
		Code:       "not_found",
		Message:    fmt.Sprintf("Environment %q was not found in project %s.", request.IDOrSlug, request.ProjectID),
	}
}

func environmentMatches(env Environment, idOrSlug string) bool {
	if idOrSlug == "" {
		return false
	}
	return strings.EqualFold(env.ID, idOrSlug) ||
		strings.EqualFold(env.Slug, idOrSlug) ||
		strings.EqualFold(env.Name, idOrSlug)
}

func IsSystemEnvironmentID(id string) bool {
	switch id {
	case EnvironmentIDProduction, EnvironmentIDPreview, EnvironmentIDDevelopment:
		return true
	default:
		return false
	}
}

func systemEnvironments(teamID, projectID string) []Environment {
	return []Environment{
		{
			ID:          EnvironmentIDProduction,
			Slug:        EnvironmentIDProduction,
			Name:        "Production",
			Type:        EnvironmentIDProduction,
			Description: "Primary project environment meant to serve qualified, promoted deployments to real users",
			Lifecycle:   EnvironmentLifecycleSystem,
			ManagedBy:   EnvironmentManagedByVercel,
			Capabilities: EnvironmentCapabilities{
				Domains:     true,
				Deployments: true,
			},
			TeamID:    teamID,
			ProjectID: projectID,
		},
		{
			ID:          EnvironmentIDPreview,
			Slug:        EnvironmentIDPreview,
			Name:        "Preview",
			Type:        EnvironmentIDPreview,
			Description: "Standard environment — included with all Vercel projects — for previewing changes before promoting them to production",
			Lifecycle:   EnvironmentLifecycleSystem,
			ManagedBy:   EnvironmentManagedByVercel,
			BranchMatcher: &BranchMatcher{
				Type:    "equals",
				Pattern: "All unassigned git branches",
			},
			Capabilities: EnvironmentCapabilities{
				Domains:     true,
				Deployments: true,
			},
			TeamID:    teamID,
			ProjectID: projectID,
		},
		{
			ID:          EnvironmentIDDevelopment,
			Slug:        EnvironmentIDDevelopment,
			Name:        "Development",
			Type:        EnvironmentIDDevelopment,
			Description: "Standard environment — included with all Vercel projects — used to supply environment variables in local development",
			Lifecycle:   EnvironmentLifecycleSystem,
			ManagedBy:   EnvironmentManagedByVercel,
			TeamID:      teamID,
			ProjectID:   projectID,
		},
	}
}

func (c customEnvironmentJSON) toEnvironment(teamID, projectID string) Environment {
	envType := c.Type
	if envType == "" {
		envType = EnvironmentIDPreview
	}
	return Environment{
		ID:            c.ID,
		Slug:          c.Slug,
		Name:          c.Slug,
		Type:          envType,
		Description:   c.Description,
		Lifecycle:     EnvironmentLifecycleCustom,
		ManagedBy:     EnvironmentManagedByUser,
		BranchMatcher: c.BranchMatcher,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
		Capabilities: EnvironmentCapabilities{
			Manage:      true,
			Domains:     true,
			Deployments: true,
		},
		TeamID:    teamID,
		ProjectID: projectID,
	}
}
