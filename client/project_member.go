package client

import (
	"context"
	"fmt"

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
}

func (c *Client) ListProjectMembers(ctx context.Context, request GetProjectMembersRequest) ([]ProjectMember, error) {
	url := fmt.Sprintf("%s/v1/projects/%s/members", c.baseURL, request.ProjectID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s&limit=100", url, c.TeamID(request.TeamID))
	}
	tflog.Info(ctx, "listing project members", map[string]any{
		"url": url,
	})

	var resp struct {
		Members []ProjectMember `json:"members"`
	}
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   string(mustMarshal(request)),
	}, &resp)
	if err != nil {
		tflog.Error(ctx, "error getting project members", map[string]any{
			"url": url,
		})
	}
	return resp.Members, err
}
