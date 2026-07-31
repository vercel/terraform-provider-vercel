package client

import (
	"context"
	"fmt"
	neturl "net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// VCRRepositoryPermission represents a permission on a Vercel Container
// Registry repository that grants another team read (pull) access to its
// images.
type VCRRepositoryPermission struct {
	RepositoryID    string `json:"repositoryId"`
	GrantedTeamID   string `json:"teamId"`
	GrantedTeamSlug string `json:"teamSlug"`
	CreatedAt       string `json:"createdAt"`
	TeamID          string `json:"-"`
}

func (c *Client) vcrRepositoryPermissionsURL(teamID, projectID, idOrName, suffix string) string {
	url := fmt.Sprintf("%s/v1/vcr/repository/%s/permissions%s?projectId=%s", c.baseURL, idOrName, suffix, projectID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s&teamId=%s", url, c.TeamID(teamID))
	}
	return url
}

type vcrRepositoryPermissionResponse struct {
	Permission VCRRepositoryPermission `json:"permission"`
}

type CreateVCRRepositoryPermissionRequest struct {
	TeamID    string `json:"-"`
	ProjectID string `json:"-"`
	IDOrName  string `json:"-"`
	// Exactly one of GrantedTeamID or GrantedTeamSlug must be set.
	GrantedTeamID   string `json:"teamId,omitempty"`
	GrantedTeamSlug string `json:"teamSlug,omitempty"`
}

func (c *Client) CreateVCRRepositoryPermission(ctx context.Context, request CreateVCRRepositoryPermissionRequest) (res VCRRepositoryPermission, err error) {
	url := c.vcrRepositoryPermissionsURL(request.TeamID, request.ProjectID, request.IDOrName, "")
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating vcr repository permission", map[string]any{
		"url":     url,
		"payload": payload,
	})
	var out vcrRepositoryPermissionResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &out)
	if err != nil {
		return res, err
	}
	res = out.Permission
	res.TeamID = c.TeamID(request.TeamID)
	return res, nil
}

type ListVCRRepositoryPermissionsRequest struct {
	TeamID    string
	ProjectID string
	IDOrName  string
}

type listVCRRepositoryPermissionsResponse struct {
	Permissions []VCRRepositoryPermission `json:"permissions"`
	NextCursor  string                    `json:"nextCursor"`
}

func (c *Client) ListVCRRepositoryPermissions(ctx context.Context, request ListVCRRepositoryPermissionsRequest) (res []VCRRepositoryPermission, err error) {
	baseURL := c.vcrRepositoryPermissionsURL(request.TeamID, request.ProjectID, request.IDOrName, "") + "&limit=100"
	cursor := ""
	for {
		url := baseURL
		if cursor != "" {
			url = fmt.Sprintf("%s&cursor=%s", url, cursor)
		}
		tflog.Info(ctx, "listing vcr repository permissions", map[string]any{
			"url": url,
		})
		var out listVCRRepositoryPermissionsResponse
		err = c.doRequest(clientRequest{
			ctx:    ctx,
			method: "GET",
			url:    url,
		}, &out)
		if err != nil {
			return res, err
		}
		for _, permission := range out.Permissions {
			permission.TeamID = c.TeamID(request.TeamID)
			res = append(res, permission)
		}
		if out.NextCursor == "" {
			return res, nil
		}
		cursor = neturl.QueryEscape(out.NextCursor)
	}
}

type GetVCRRepositoryPermissionRequest struct {
	TeamID    string
	ProjectID string
	IDOrName  string
	// A permission matches if either GrantedTeamID or GrantedTeamSlug does.
	GrantedTeamID   string
	GrantedTeamSlug string
}

// GetVCRRepositoryPermission finds a single repository permission by the team
// it was granted to. The API has no individual GET, so this lists and filters.
func (c *Client) GetVCRRepositoryPermission(ctx context.Context, request GetVCRRepositoryPermissionRequest) (res VCRRepositoryPermission, err error) {
	permissions, err := c.ListVCRRepositoryPermissions(ctx, ListVCRRepositoryPermissionsRequest{
		TeamID:    request.TeamID,
		ProjectID: request.ProjectID,
		IDOrName:  request.IDOrName,
	})
	if err != nil {
		return res, err
	}
	for _, permission := range permissions {
		if (request.GrantedTeamID != "" && permission.GrantedTeamID == request.GrantedTeamID) ||
			(request.GrantedTeamSlug != "" && permission.GrantedTeamSlug == request.GrantedTeamSlug) {
			return permission, nil
		}
	}
	grantedTeam := request.GrantedTeamID
	if grantedTeam == "" {
		grantedTeam = request.GrantedTeamSlug
	}
	return res, APIError{
		StatusCode: 404,
		Code:       "not_found",
		Message:    fmt.Sprintf("The repository %s is not shared with team %s.", request.IDOrName, grantedTeam),
	}
}

type DeleteVCRRepositoryPermissionRequest struct {
	TeamID    string `json:"-"`
	ProjectID string `json:"-"`
	IDOrName  string `json:"-"`
	// Exactly one of GrantedTeamID or GrantedTeamSlug must be set.
	GrantedTeamID   string `json:"teamId,omitempty"`
	GrantedTeamSlug string `json:"teamSlug,omitempty"`
}

func (c *Client) DeleteVCRRepositoryPermission(ctx context.Context, request DeleteVCRRepositoryPermissionRequest) error {
	url := c.vcrRepositoryPermissionsURL(request.TeamID, request.ProjectID, request.IDOrName, "")
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "deleting vcr repository permission", map[string]any{
		"url":     url,
		"payload": payload,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   payload,
	}, nil)
}

type DeleteAllVCRRepositoryPermissionsRequest struct {
	TeamID    string
	ProjectID string
	IDOrName  string
}

// DeleteAllVCRRepositoryPermissions removes every permission from a
// repository, so it is no longer shared with any team.
func (c *Client) DeleteAllVCRRepositoryPermissions(ctx context.Context, request DeleteAllVCRRepositoryPermissionsRequest) error {
	url := c.vcrRepositoryPermissionsURL(request.TeamID, request.ProjectID, request.IDOrName, "/all")
	tflog.Info(ctx, "deleting all vcr repository permissions", map[string]any{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}
