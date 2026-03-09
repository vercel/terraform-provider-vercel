package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// BulkRedirect represents a single project-level redirect.
type BulkRedirect struct {
	Source              string `json:"source"`
	Destination         string `json:"destination"`
	StatusCode          *int64 `json:"statusCode,omitempty"`
	Permanent           *bool  `json:"permanent,omitempty"`
	CaseSensitive       *bool  `json:"caseSensitive,omitempty"`
	Query               *bool  `json:"query,omitempty"`
	PreserveQueryParams *bool  `json:"preserveQueryParams,omitempty"`
}

// BulkRedirectVersion represents a bulk redirects version.
type BulkRedirectVersion struct {
	ID            string  `json:"id"`
	Key           string  `json:"key"`
	LastModified  int64   `json:"lastModified"`
	CreatedBy     string  `json:"createdBy"`
	Name          *string `json:"name"`
	IsStaging     bool    `json:"isStaging"`
	IsLive        bool    `json:"isLive"`
	RedirectCount int64   `json:"redirectCount"`
	Alias         *string `json:"alias"`
}

// BulkRedirects represents the redirects for a given project and version.
type BulkRedirects struct {
	ProjectID string               `json:"-"`
	TeamID    string               `json:"-"`
	Redirects []BulkRedirect       `json:"redirects"`
	Version   *BulkRedirectVersion `json:"version,omitempty"`
}

type bulkRedirectsPagination struct {
	Page     int `json:"page"`
	PerPage  int `json:"per_page"`
	NumPages int `json:"numPages"`
}

type bulkRedirectsResponse struct {
	Redirects  []BulkRedirect          `json:"redirects"`
	Version    *BulkRedirectVersion    `json:"version"`
	Pagination bulkRedirectsPagination `json:"pagination"`
}

type bulkRedirectVersionsResponse struct {
	Versions []BulkRedirectVersion `json:"versions"`
}

type bulkRedirectVersionMutationResponse struct {
	Alias   *string             `json:"alias"`
	Version BulkRedirectVersion `json:"version"`
}

// GetBulkRedirectsRequest defines the information required to read redirects for a project.
type GetBulkRedirectsRequest struct {
	ProjectID string
	TeamID    string
	VersionID string
}

func (c *Client) bulkRedirectsURL(path string, teamID string, query url.Values) string {
	if query == nil {
		query = url.Values{}
	}

	if resolvedTeamID := c.TeamID(teamID); resolvedTeamID != "" && query.Get("teamId") == "" {
		query.Set("teamId", resolvedTeamID)
	}

	if encoded := query.Encode(); encoded != "" {
		return fmt.Sprintf("%s%s?%s", c.baseURL, path, encoded)
	}

	return fmt.Sprintf("%s%s", c.baseURL, path)
}

// GetBulkRedirects reads the redirects for a project version, following pagination until all redirects are returned.
func (c *Client) GetBulkRedirects(ctx context.Context, request GetBulkRedirectsRequest) (BulkRedirects, error) {
	result := BulkRedirects{
		ProjectID: request.ProjectID,
		TeamID:    c.TeamID(request.TeamID),
		Redirects: []BulkRedirect{},
	}

	for page := 1; ; page++ {
		query := url.Values{}
		query.Set("projectId", request.ProjectID)
		query.Set("page", strconv.Itoa(page))
		query.Set("per_page", "250")
		if request.VersionID != "" {
			query.Set("versionId", request.VersionID)
		}

		url := c.bulkRedirectsURL("/v1/bulk-redirects", request.TeamID, query)
		tflog.Info(ctx, "reading bulk redirects", map[string]any{
			"page":       page,
			"project_id": request.ProjectID,
			"team_id":    c.TeamID(request.TeamID),
			"url":        url,
			"version_id": request.VersionID,
		})

		var response bulkRedirectsResponse
		err := c.doRequest(clientRequest{
			ctx:    ctx,
			method: "GET",
			url:    url,
		}, &response)
		if err != nil {
			return BulkRedirects{}, err
		}

		result.Redirects = append(result.Redirects, response.Redirects...)
		if response.Version != nil {
			result.Version = response.Version
		}

		if response.Pagination.NumPages == 0 || page >= response.Pagination.NumPages {
			return result, nil
		}
	}
}

// GetBulkRedirectVersions reads the version history for a project's bulk redirects.
func (c *Client) GetBulkRedirectVersions(ctx context.Context, projectID, teamID string) ([]BulkRedirectVersion, error) {
	query := url.Values{}
	query.Set("projectId", projectID)

	url := c.bulkRedirectsURL("/v1/bulk-redirects/versions", teamID, query)
	tflog.Info(ctx, "reading bulk redirect versions", map[string]any{
		"project_id": projectID,
		"team_id":    c.TeamID(teamID),
		"url":        url,
	})

	var response bulkRedirectVersionsResponse
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &response)
	if err != nil {
		return nil, err
	}

	return response.Versions, nil
}

// StageBulkRedirectsRequest defines the information required to stage a full redirects set for a project.
type StageBulkRedirectsRequest struct {
	ProjectID   string         `json:"projectId"`
	TeamID      string         `json:"teamId,omitempty"`
	Overwrite   bool           `json:"overwrite"`
	Redirects   []BulkRedirect `json:"redirects"`
	VersionName *string        `json:"name,omitempty"`
}

// StageBulkRedirects stages a full redirects set for a project.
func (c *Client) StageBulkRedirects(ctx context.Context, request StageBulkRedirectsRequest) (BulkRedirectVersion, error) {
	request.TeamID = c.TeamID(request.TeamID)

	url := c.bulkRedirectsURL("/v1/bulk-redirects", request.TeamID, nil)
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "staging bulk redirects", map[string]any{
		"payload":    payload,
		"project_id": request.ProjectID,
		"team_id":    request.TeamID,
		"url":        url,
	})

	var response bulkRedirectVersionMutationResponse
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PUT",
		url:    url,
		body:   payload,
	}, &response)
	if err != nil {
		return BulkRedirectVersion{}, err
	}

	if response.Version.Alias == nil {
		response.Version.Alias = response.Alias
	}

	return response.Version, nil
}

// UpdateBulkRedirectVersionRequest defines the information required to promote or restore a redirects version.
type UpdateBulkRedirectVersionRequest struct {
	ProjectID   string  `json:"-"`
	TeamID      string  `json:"-"`
	VersionID   string  `json:"id"`
	Action      string  `json:"action"`
	VersionName *string `json:"name,omitempty"`
}

// UpdateBulkRedirectVersion promotes or restores a redirects version.
func (c *Client) UpdateBulkRedirectVersion(ctx context.Context, request UpdateBulkRedirectVersionRequest) (BulkRedirectVersion, error) {
	query := url.Values{}
	query.Set("projectId", request.ProjectID)
	url := c.bulkRedirectsURL("/v1/bulk-redirects/versions", request.TeamID, query)

	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating bulk redirects version", map[string]any{
		"action":     request.Action,
		"payload":    payload,
		"project_id": request.ProjectID,
		"team_id":    c.TeamID(request.TeamID),
		"url":        url,
		"version_id": request.VersionID,
	})

	var response bulkRedirectVersionMutationResponse
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &response)
	if err != nil {
		return BulkRedirectVersion{}, err
	}

	if response.Version.Alias == nil {
		response.Version.Alias = response.Alias
	}

	return response.Version, nil
}
