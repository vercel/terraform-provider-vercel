package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type UserToken struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Prefix      *string `json:"prefix"`
	Suffix      *string `json:"suffix"`
	Origin      *string `json:"origin"`
	TeamID      *string `json:"teamId"`
	CreatedAt   int64   `json:"createdAt"`
	ActiveAt    int64   `json:"activeAt"`
	ExpiresAt   *int64  `json:"expiresAt"`
	LeakedAt    *int64  `json:"leakedAt"`
	LeakedURL   *string `json:"leakedUrl"`
	ProjectID   *string `json:"projectId"`
	BearerToken *string
}

type CreateUserTokenRequest struct {
	Name      string  `json:"name"`
	ExpiresAt *int64  `json:"expiresAt,omitempty"`
	ProjectID *string `json:"projectId,omitempty"`
	TeamID    string  `json:"-"`
}

func (c *Client) CreateUserToken(ctx context.Context, request CreateUserTokenRequest) (u UserToken, err error) {
	url := fmt.Sprintf("%s/v3/user/tokens", c.baseURL)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating user token", map[string]any{
		"url":     url,
		"payload": payload,
	})

	var response struct {
		Token       UserToken `json:"token"`
		BearerToken string    `json:"bearerToken"`
	}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &response)
	if response.BearerToken != "" {
		response.Token.BearerToken = &response.BearerToken
	}
	return response.Token, err
}

func (c *Client) GetUserToken(ctx context.Context, tokenID string) (u UserToken, err error) {
	url := fmt.Sprintf("%s/v5/user/tokens/%s", c.baseURL, tokenID)
	tflog.Info(ctx, "getting user token", map[string]any{
		"url": url,
	})

	var response struct {
		Token UserToken `json:"token"`
	}
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &response)
	return response.Token, err
}

func (c *Client) DeleteUserToken(ctx context.Context, tokenID, teamID string) error {
	url := fmt.Sprintf("%s/v3/user/tokens/%s", c.baseURL, tokenID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "deleting user token", map[string]any{
		"url": url,
	})

	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}
