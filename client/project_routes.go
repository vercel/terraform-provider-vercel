package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type ProjectRouteCondition struct {
	Type  string  `json:"type"`
	Key   *string `json:"key,omitempty"`
	Value *string `json:"value,omitempty"`
}

type ProjectRouteTransform struct {
	Type   string   `json:"type"`
	Op     string   `json:"op"`
	Target any      `json:"target,omitempty"`
	Args   any      `json:"args,omitempty"`
	Env    []string `json:"env,omitempty"`
}

type ProjectRouteDefinition struct {
	Src                       string                  `json:"src"`
	Dest                      *string                 `json:"dest,omitempty"`
	Headers                   map[string]string       `json:"headers,omitempty"`
	CaseSensitive             *bool                   `json:"caseSensitive,omitempty"`
	Status                    *int64                  `json:"status,omitempty"`
	Has                       []ProjectRouteCondition `json:"has,omitempty"`
	Missing                   []ProjectRouteCondition `json:"missing,omitempty"`
	Transforms                []ProjectRouteTransform `json:"transforms,omitempty"`
	RespectOriginCacheControl *bool                   `json:"respectOriginCacheControl,omitempty"`
}

type ProjectRoutingRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description *string                `json:"description,omitempty"`
	Enabled     *bool                  `json:"enabled,omitempty"`
	Route       ProjectRouteDefinition `json:"route"`
	RawSrc      *string                `json:"rawSrc,omitempty"`
	RawDest     *string                `json:"rawDest,omitempty"`
	SrcSyntax   *string                `json:"srcSyntax,omitempty"`
	RouteType   *string                `json:"routeType,omitempty"`
}

type ProjectRouteVersion struct {
	ID           string  `json:"id"`
	S3Key        string  `json:"s3Key"`
	LastModified int64   `json:"lastModified"`
	CreatedBy    string  `json:"createdBy"`
	IsStaging    bool    `json:"isStaging"`
	IsLive       bool    `json:"isLive"`
	RuleCount    int64   `json:"ruleCount"`
	Alias        *string `json:"alias,omitempty"`
}

type ProjectRouteLimit struct {
	MaxRoutes     int64 `json:"maxRoutes"`
	CurrentRoutes int64 `json:"currentRoutes"`
}

type ProjectRoutingRulesResponse struct {
	Routes  []ProjectRoutingRule `json:"routes"`
	Version ProjectRouteVersion  `json:"version"`
	Limit   ProjectRouteLimit    `json:"limit"`
}

type projectRouteVersionsResponse struct {
	Versions []ProjectRouteVersion `json:"versions"`
}

type ProjectRoutingRuleInput struct {
	Name        string                 `json:"name"`
	Description *string                `json:"description,omitempty"`
	Enabled     *bool                  `json:"enabled,omitempty"`
	SrcSyntax   *string                `json:"srcSyntax,omitempty"`
	Route       ProjectRouteDefinition `json:"route"`
}

type ProjectRoutingRulePosition struct {
	Placement   string  `json:"placement"`
	ReferenceID *string `json:"referenceId,omitempty"`
}

type AddProjectRouteRequest struct {
	TeamID    string                      `json:"-"`
	ProjectID string                      `json:"-"`
	Route     ProjectRoutingRuleInput     `json:"route"`
	Position  *ProjectRoutingRulePosition `json:"position,omitempty"`
}

type EditProjectRouteRequest struct {
	TeamID    string                   `json:"-"`
	ProjectID string                   `json:"-"`
	RouteID   string                   `json:"-"`
	Route     *ProjectRoutingRuleInput `json:"route,omitempty"`
	Restore   bool                     `json:"restore,omitempty"`
}

type ProjectRouteMutationResponse struct {
	Route   ProjectRoutingRule  `json:"route"`
	Version ProjectRouteVersion `json:"version"`
}

type UpdateProjectRoutingRuleVersionRequest struct {
	TeamID    string `json:"-"`
	ProjectID string `json:"-"`
	ID        string `json:"id"`
	Action    string `json:"action"`
}

type updateProjectRoutingRuleVersionResponse struct {
	Version ProjectRouteVersion `json:"version"`
}

type DeleteProjectRoutesRequest struct {
	TeamID    string   `json:"-"`
	ProjectID string   `json:"-"`
	RouteIDs  []string `json:"routeIds"`
}

type DeleteProjectRoutesResponse struct {
	DeletedCount int64               `json:"deletedCount"`
	Version      ProjectRouteVersion `json:"version"`
}

func (c *Client) GetProjectRoutingRules(ctx context.Context, projectID, teamID, versionID string) (r ProjectRoutingRulesResponse, err error) {
	urlStr := fmt.Sprintf("%s/v1/projects/%s/routes", c.baseURL, projectID)

	query := url.Values{}
	if c.TeamID(teamID) != "" {
		query.Set("teamId", c.TeamID(teamID))
	}
	if versionID != "" {
		query.Set("versionId", versionID)
	}
	if encodedQuery := query.Encode(); encodedQuery != "" {
		urlStr = fmt.Sprintf("%s?%s", urlStr, encodedQuery)
	}

	tflog.Info(ctx, "getting project routing rules", map[string]any{
		"url": urlStr,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    urlStr,
	}, &r)
	if err != nil {
		return r, fmt.Errorf("unable to get project routing rules: %w", err)
	}

	return r, nil
}

func (c *Client) GetProjectRouteVersions(ctx context.Context, projectID, teamID string) (versions []ProjectRouteVersion, err error) {
	urlStr := fmt.Sprintf("%s/v1/projects/%s/routes/versions", c.baseURL, projectID)
	if c.TeamID(teamID) != "" {
		urlStr = fmt.Sprintf("%s?teamId=%s", urlStr, c.TeamID(teamID))
	}

	tflog.Info(ctx, "getting project routing rule versions", map[string]any{
		"url": urlStr,
	})

	var response projectRouteVersionsResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    urlStr,
	}, &response)
	if err != nil {
		return nil, fmt.Errorf("unable to get project routing rule versions: %w", err)
	}

	return response.Versions, nil
}

func (c *Client) AddProjectRoute(ctx context.Context, request AddProjectRouteRequest) (response ProjectRouteMutationResponse, err error) {
	urlStr := fmt.Sprintf("%s/v1/projects/%s/routes", c.baseURL, request.ProjectID)
	if c.TeamID(request.TeamID) != "" {
		urlStr = fmt.Sprintf("%s?teamId=%s", urlStr, c.TeamID(request.TeamID))
	}

	payload := string(mustMarshal(request))
	tflog.Info(ctx, "adding project routing rule", map[string]any{
		"url":     urlStr,
		"payload": payload,
	})

	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    urlStr,
		body:   payload,
	}, &response)
	if err != nil {
		return response, fmt.Errorf("unable to add project routing rule: %w", err)
	}

	return response, nil
}

func (c *Client) EditProjectRoute(ctx context.Context, request EditProjectRouteRequest) (response ProjectRouteMutationResponse, err error) {
	urlStr := fmt.Sprintf("%s/v1/projects/%s/routes/%s", c.baseURL, request.ProjectID, request.RouteID)
	if c.TeamID(request.TeamID) != "" {
		urlStr = fmt.Sprintf("%s?teamId=%s", urlStr, c.TeamID(request.TeamID))
	}

	payload := string(mustMarshal(request))
	tflog.Info(ctx, "editing project routing rule", map[string]any{
		"url":     urlStr,
		"payload": payload,
	})

	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    urlStr,
		body:   payload,
	}, &response)
	if err != nil {
		return response, fmt.Errorf("unable to edit project routing rule: %w", err)
	}

	return response, nil
}

func (c *Client) DeleteProjectRoutes(ctx context.Context, request DeleteProjectRoutesRequest) (response DeleteProjectRoutesResponse, err error) {
	urlStr := fmt.Sprintf("%s/v1/projects/%s/routes", c.baseURL, request.ProjectID)
	if c.TeamID(request.TeamID) != "" {
		urlStr = fmt.Sprintf("%s?teamId=%s", urlStr, c.TeamID(request.TeamID))
	}

	payload := string(mustMarshal(request))
	tflog.Info(ctx, "deleting project routing rules", map[string]any{
		"url":     urlStr,
		"payload": payload,
	})

	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    urlStr,
		body:   payload,
	}, &response)
	if err != nil {
		return response, fmt.Errorf("unable to delete project routing rules: %w", err)
	}

	return response, nil
}

func (c *Client) UpdateProjectRoutingRuleVersion(ctx context.Context, request UpdateProjectRoutingRuleVersionRequest) (version ProjectRouteVersion, err error) {
	urlStr := fmt.Sprintf("%s/v1/projects/%s/routes/versions", c.baseURL, request.ProjectID)
	if c.TeamID(request.TeamID) != "" {
		urlStr = fmt.Sprintf("%s?teamId=%s", urlStr, c.TeamID(request.TeamID))
	}

	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating project routing rule version", map[string]any{
		"url":     urlStr,
		"payload": payload,
	})

	var response updateProjectRoutingRuleVersionResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    urlStr,
		body:   payload,
	}, &response)
	if err != nil {
		return version, fmt.Errorf("unable to update project routing rule version: %w", err)
	}

	return response.Version, nil
}
