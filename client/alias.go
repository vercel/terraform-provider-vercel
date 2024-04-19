package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// CreateAliasRequest defines the request the Vercel API expects in order to create an alias.
type CreateAliasRequest struct {
	Alias string `json:"alias"`
}

// The create Alias endpoint does not return the full AliasResponse, only the UID and Alias.
type createAliasResponse struct {
	UID    string `json:"uid"`
	Alias  string `json:"alias"`
	TeamID string `json:"-"`
}

// CreateAlias creates an alias within Vercel.
func (c *Client) CreateAlias(ctx context.Context, request CreateAliasRequest, deploymentID string, teamID string) (r AliasResponse, err error) {
	url := fmt.Sprintf("%s/v2/deployments/%s/aliases", c.baseURL, deploymentID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	payload := string(mustMarshal(request))

	tflog.Info(ctx, "creating alias", map[string]interface{}{
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
		DeploymentID: deploymentID,
		TeamID:       c.teamID(teamID),
	}, nil
}

// DeleteAliasResponse defines the response the Vercel API returns when an alias is deleted.
type DeleteAliasResponse struct {
	Status string `json:"status"`
}

// DeleteAlias deletes an alias within Vercel.
func (c *Client) DeleteAlias(ctx context.Context, aliasUID string, teamID string) (r DeleteAliasResponse, err error) {
	url := fmt.Sprintf("%s/v2/aliases/%s", c.baseURL, aliasUID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	tflog.Info(ctx, "deleting alias", map[string]interface{}{
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
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	tflog.Info(ctx, "getting alias", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &r)
	r.TeamID = c.teamID(teamID)
	return r, err
}
