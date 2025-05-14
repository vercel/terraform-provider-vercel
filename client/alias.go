package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// UpsertAliasRequest defines the request the Vercel API expects in order to create an alias.
type UpsertAliasRequest struct {
	Alias        string `json:"alias"`
	DeploymentID string `json:"-"`
	TeamID       string `json:"-"`
}

// The create Alias endpoint does not return the full AliasResponse, only the UID and Alias.
type createAliasResponse struct {
	UID   string `json:"uid"`
	Alias string `json:"alias"`
}

// UpsertAlias creates an alias within Vercel.
func (c *Client) UpsertAlias(ctx context.Context, request UpsertAliasRequest) (r AliasResponse, err error) {
	url := fmt.Sprintf("%s/v2/deployments/%s/aliases", c.baseURL, request.DeploymentID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	payload := string(mustMarshal(request))

	tflog.Info(ctx, "creating alias", map[string]any{
		"url":     url,
		"payload": payload,
	})
	var aliasResponse createAliasResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &aliasResponse)
	if err != nil {
		return r, err
	}

	return AliasResponse{
		UID:          aliasResponse.UID,
		Alias:        aliasResponse.Alias,
		DeploymentID: request.DeploymentID,
		TeamID:       c.TeamID(request.TeamID),
	}, nil
}

// DeleteAliasResponse defines the response the Vercel API returns when an alias is deleted.
type DeleteAliasResponse struct {
	Status string `json:"status"`
}

// DeleteAlias deletes an alias within Vercel.
func (c *Client) DeleteAlias(ctx context.Context, aliasUID string, teamID string) (r DeleteAliasResponse, err error) {
	url := fmt.Sprintf("%s/v2/aliases/%s", c.baseURL, aliasUID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}

	tflog.Info(ctx, "deleting alias", map[string]any{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
		body:   "",
	}, &r)
	return r, err
}

// AliasResponse defines the response the Vercel API returns for an alias.
type AliasResponse struct {
	UID          string `json:"uid"`
	Alias        string `json:"alias"`
	DeploymentID string `json:"deploymentId"`
	TeamID       string `json:"-"`
}

// GetAlias retrieves information about an existing alias from vercel.
func (c *Client) GetAlias(ctx context.Context, alias, teamID string) (r AliasResponse, err error) {
	url := fmt.Sprintf("%s/v4/aliases/%s", c.baseURL, alias)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "getting alias", map[string]any{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &r)
	r.TeamID = c.TeamID(teamID)
	return r, err
}
