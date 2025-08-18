package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type ProjectRole struct {
	ProjectID string `json:"projectId"`
	Role      string `json:"role"`
}

type TeamMemberInviteRequest struct {
	UserID       string        `json:"uid,omitempty"` // Deprecated: UserID is no longer supported by Vercel API
	Email        string        `json:"email,omitempty"`
	Role         string        `json:"role,omitempty"`
	Projects     []ProjectRole `json:"projects,omitempty"`
	AccessGroups []string      `json:"accessGroups,omitempty"`
	TeamID       string        `json:"-"`
}

type TeamMemberInviteResponse struct {
	UserID   string `json:"uid"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func (c *Client) InviteTeamMember(ctx context.Context, request TeamMemberInviteRequest) (TeamMemberInviteResponse, error) {
	url := fmt.Sprintf("%s/v1/teams/%s/members", c.baseURL, request.TeamID)
	tflog.Info(ctx, "inviting user", map[string]any{
		"url":   url,
		"user":  request.UserID,
		"email": request.Email,
		"role":  request.Role,
	})

	var res TeamMemberInviteResponse
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   string(mustMarshal(request)),
	}, &res)
	return res, err
}

type TeamMemberRemoveRequest struct {
	UserID string
	TeamID string
}

func (c *Client) RemoveTeamMember(ctx context.Context, request TeamMemberRemoveRequest) error {
	url := fmt.Sprintf("%s/v2/teams/%s/members/%s", c.baseURL, request.TeamID, request.UserID)
	tflog.Info(ctx, "removing user", map[string]any{
		"url":  url,
		"user": request.UserID,
	})
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   "",
	}, nil)
	return err
}

type TeamMemberUpdateRequest struct {
	UserID               string        `json:"-"`
	Role                 string        `json:"role"`
	TeamID               string        `json:"-"`
	Projects             []ProjectRole `json:"projects,omitempty"`
	AccessGroupsToAdd    []string      `json:"accessGroupsToAdd,omitempty"`
	AccessGroupsToRemove []string      `json:"accessGroupsToRemove,omitempty"`
}

func (c *Client) UpdateTeamMember(ctx context.Context, request TeamMemberUpdateRequest) error {
	url := fmt.Sprintf("%s/v1/teams/%s/members/%s", c.baseURL, request.TeamID, request.UserID)
	tflog.Info(ctx, "updating team member", map[string]any{
		"url":  url,
		"user": request.UserID,
		"role": request.Role,
	})
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   string(mustMarshal(request)),
	}, nil)
	return err
}

type GetTeamMemberRequest struct {
	TeamID string
	UserID string
}

type TeamMember struct {
	Confirmed    bool          `json:"confirmed"`
	Role         string        `json:"role"`
	UserID       string        `json:"uid"`
	Email        string        `json:"email"`
	Projects     []ProjectRole `json:"projects"`
	AccessGroups []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"accessGroups"`
}

func (c *Client) GetTeamMember(ctx context.Context, request GetTeamMemberRequest) (TeamMember, error) {
	url := fmt.Sprintf("%s/v2/teams/%s/members?limit=1&filterByUserIds=%s", c.baseURL, request.TeamID, request.UserID)
	tflog.Info(ctx, "getting team member", map[string]any{
		"url": url,
	})

	var response struct {
		Members []TeamMember `json:"members"`
	}
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &response)
	if err != nil {
		return TeamMember{}, err
	}
	if len(response.Members) == 0 {
		return TeamMember{}, APIError{
			StatusCode: 404,
			Message:    "Team member not found",
			Code:       "not_found",
		}
	}

	// Now look up the projects for the member, but only if we need to.
	if !response.Members[0].Confirmed || (response.Members[0].Role != "DEVELOPER" && response.Members[0].Role != "CONTRIBUTOR") {
		return response.Members[0], nil
	}
	url = fmt.Sprintf("%s/v1/teams/%s/members/%s/projects?limit=100", c.baseURL, request.TeamID, request.UserID)
	var response2 struct {
		Projects []ProjectRole `json:"projects"`
	}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &response2)
	if err != nil {
		return TeamMember{}, err
	}
	response.Members[0].Projects = response2.Projects
	return response.Members[0], err
}
