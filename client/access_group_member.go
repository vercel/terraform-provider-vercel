package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// AccessGroupMember represents the membership of a single user in an access group.
type AccessGroupMember struct {
	TeamID        string
	AccessGroupID string
	UserID        string
}

type CreateAccessGroupMemberRequest struct {
	TeamID        string
	AccessGroupID string
	UserID        string
}

// CreateAccessGroupMember adds a user to an access group. The Vercel API manages
// access group membership through the access group update endpoint using the
// membersToAdd field.
func (c *Client) CreateAccessGroupMember(ctx context.Context, req CreateAccessGroupMemberRequest) (r AccessGroupMember, err error) {
	url := fmt.Sprintf("%s/v1/access-groups/%s", c.baseURL, req.AccessGroupID)
	if c.TeamID(req.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(req.TeamID))
	}
	payload := string(mustMarshal(
		struct {
			MembersToAdd []string `json:"membersToAdd"`
		}{
			MembersToAdd: []string{req.UserID},
		},
	))
	tflog.Info(ctx, "creating access group member", map[string]any{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, nil)
	if err != nil {
		return r, err
	}
	return AccessGroupMember{
		TeamID:        c.TeamID(req.TeamID),
		AccessGroupID: req.AccessGroupID,
		UserID:        req.UserID,
	}, nil
}

type GetAccessGroupMemberRequest struct {
	TeamID        string
	AccessGroupID string
	UserID        string
}

// accessGroupMembersResponse models the paginated response from the list
// members endpoint.
type accessGroupMembersResponse struct {
	Members []struct {
		UID string `json:"uid"`
	} `json:"members"`
	Pagination struct {
		Next *string `json:"next"`
	} `json:"pagination"`
}

// GetAccessGroupMember looks up a single member of an access group. The Vercel
// API does not expose a single-member endpoint, so this pages through the list
// members endpoint searching for the requested user. A 404 APIError is returned
// when the user is not a member, so callers can use NotFound to detect removal.
func (c *Client) GetAccessGroupMember(ctx context.Context, req GetAccessGroupMemberRequest) (r AccessGroupMember, err error) {
	next := ""
	for {
		url := fmt.Sprintf("%s/v1/access-groups/%s/members?limit=100", c.baseURL, req.AccessGroupID)
		if c.TeamID(req.TeamID) != "" {
			url = fmt.Sprintf("%s&teamId=%s", url, c.TeamID(req.TeamID))
		}
		if next != "" {
			url = fmt.Sprintf("%s&next=%s", url, next)
		}
		tflog.Info(ctx, "getting access group member", map[string]any{
			"url": url,
		})
		var response accessGroupMembersResponse
		err = c.doRequest(clientRequest{
			ctx:    ctx,
			method: "GET",
			url:    url,
		}, &response)
		if err != nil {
			return r, fmt.Errorf("unable to get access group member: %w", err)
		}

		for _, m := range response.Members {
			if m.UID == req.UserID {
				return AccessGroupMember{
					TeamID:        c.TeamID(req.TeamID),
					AccessGroupID: req.AccessGroupID,
					UserID:        req.UserID,
				}, nil
			}
		}

		if response.Pagination.Next == nil || *response.Pagination.Next == "" {
			break
		}
		next = *response.Pagination.Next
	}

	return r, APIError{
		StatusCode: 404,
		Message:    "Access group member not found",
		Code:       "not_found",
	}
}

type DeleteAccessGroupMemberRequest struct {
	TeamID        string
	AccessGroupID string
	UserID        string
}

// DeleteAccessGroupMember removes a user from an access group via the access
// group update endpoint using the membersToRemove field.
func (c *Client) DeleteAccessGroupMember(ctx context.Context, req DeleteAccessGroupMemberRequest) error {
	url := fmt.Sprintf("%s/v1/access-groups/%s", c.baseURL, req.AccessGroupID)
	if c.TeamID(req.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(req.TeamID))
	}
	payload := string(mustMarshal(
		struct {
			MembersToRemove []string `json:"membersToRemove"`
		}{
			MembersToRemove: []string{req.UserID},
		},
	))
	tflog.Info(ctx, "deleting access group member", map[string]any{
		"url":     url,
		"payload": payload,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, nil)
}
