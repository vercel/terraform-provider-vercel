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
