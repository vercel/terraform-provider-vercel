package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type ProjectMember struct {
	UserID   string `json:"uid,omitempty"`
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
	Role     string `json:"role"`
}

type AddProjectMembersRequest struct {
	ProjectID string          `json:"-"`
	TeamID    string          `json:"-"`
	Members   []ProjectMember `json:"members"`
}

func (c *Client) AddProjectMembers(ctx context.Context, request AddProjectMembersRequest) error {
	url := fmt.Sprintf("%s/v1/projects/%s/members/batch", c.baseURL, request.ProjectID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	tflog.Info(ctx, "adding project members", map[string]any{
		"url": url,
	})
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   string(mustMarshal(request)),
	}, nil)
	if err != nil {
		tflog.Error(ctx, "error adding project members", map[string]any{
			"url":     url,
			"members": request.Members,
		})
	}
	return err
}

type RemoveProjectMembersRequest struct {
	ProjectID string   `json:"-"`
	TeamID    string   `json:"-"`
	Members   []string `json:"members"`
}

func (c *Client) RemoveProjectMembers(ctx context.Context, request RemoveProjectMembersRequest) error {
	url := fmt.Sprintf("%s/v1/projects/%s/members/batch", c.baseURL, request.ProjectID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	tflog.Info(ctx, "removing project members", map[string]any{
		"url": url,
	})
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   string(mustMarshal(request)),
	}, nil)
	if err != nil {
		tflog.Error(ctx, "error removing project members", map[string]any{
			"url":     url,
			"members": request.Members,
		})
	}
	return err
}

type UpdateProjectMemberRequest struct {
	UserID string `json:"uid,omitempty"`
	Role   string `json:"role"`
}

type UpdateProjectMembersRequest struct {
	ProjectID string                       `json:"-"`
	TeamID    string                       `json:"-"`
	Members   []UpdateProjectMemberRequest `json:"members"`
}

func (c *Client) UpdateProjectMembers(ctx context.Context, request UpdateProjectMembersRequest) error {
	url := fmt.Sprintf("%s/v1/projects/%s/members", c.baseURL, request.ProjectID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating project members", map[string]any{
		"url":     url,
		"payload": payload,
	})
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, nil)
	if err != nil {
		tflog.Error(ctx, "error updating project members", map[string]any{
			"url":     url,
			"members": request.Members,
		})
	}
	return err
}

type GetProjectMembersRequest struct {
	ProjectID string `json:"-"`
	TeamID    string `json:"-"`
	Limit     int    `json:"-"`
	Until     *int64 `json:"-"`
	Since     *int64 `json:"-"`
}

type ListProjectMembersResponse struct {
	Members    []ProjectMember
	Pagination PageInfo
}

func (c *Client) ListProjectMembersPage(ctx context.Context, request GetProjectMembersRequest) (ListProjectMembersResponse, error) {
	baseURL := fmt.Sprintf("%s/v1/projects/%s/members", c.baseURL, request.ProjectID)
	query := url.Values{}
	if c.TeamID(request.TeamID) != "" {
		query.Set("teamId", c.TeamID(request.TeamID))
	}
	url := urlWithQuery(baseURL, paginationQuery(query, request.Limit, request.Until, request.Since))

	tflog.Info(ctx, "listing project members page", map[string]any{
		"url": url,
	})

	var resp struct {
		Members    []ProjectMember `json:"members"`
		Pagination PageInfo        `json:"pagination"`
	}
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &resp)
	if err != nil {
		tflog.Error(ctx, "error getting project members", map[string]any{
			"url": url,
		})
	}
	return ListProjectMembersResponse{
		Members:    resp.Members,
		Pagination: resp.Pagination,
	}, err
}

func (c *Client) ListProjectMembers(ctx context.Context, request GetProjectMembersRequest) ([]ProjectMember, error) {
	return collectPages(func(until *int64) ([]ProjectMember, PageInfo, error) {
		request.Limit = defaultPaginationLimit
		request.Until = until
		request.Since = nil
		response, err := c.ListProjectMembersPage(ctx, request)
		return response.Members, response.Pagination, err
	})
}
