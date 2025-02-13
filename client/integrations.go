package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type IntegrationProjectAccess struct {
	Allowed bool
	TeamID  string
}

func (c *Client) GetIntegrationProjectAccess(ctx context.Context, integrationID, projectID, teamID string) (IntegrationProjectAccess, error) {
	url := fmt.Sprintf("%s/v1/integrations/configuration/%s/project/%s", c.baseURL, integrationID, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	tflog.Info(ctx, "getting integration project access", map[string]interface{}{
		"url": url,
	})

	type resp struct {
		Allowed bool `json:"allowed"`
	}

	var e resp
	if err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &e); err != nil {
		return IntegrationProjectAccess{
			Allowed: false,
			TeamID:  c.teamID(teamID),
		}, err
	}
	return IntegrationProjectAccess{
		Allowed: e.Allowed,
		TeamID:  c.teamID(teamID),
	}, nil
}

func (c *Client) GrantIntegrationProjectAccess(ctx context.Context, integrationID, projectID, teamID string) (IntegrationProjectAccess, error) {
	url := fmt.Sprintf("%s/v1/integrations/configuration/%s/project/%s", c.baseURL, integrationID, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	tflog.Info(ctx, "getting integration project access", map[string]interface{}{
		"url": url,
	})

	type resp struct {
		Allowed bool `json:"allowed"`
	}

	var e resp
	if err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   `{ "allowed": true }`,
	}, &e); err != nil {
		return IntegrationProjectAccess{
			Allowed: false,
			TeamID:  c.teamID(teamID),
		}, err
	}
	return IntegrationProjectAccess{
		Allowed: true,
		TeamID:  c.teamID(teamID),
	}, nil
}

func (c *Client) RevokeIntegrationProjectAccess(ctx context.Context, integrationID, projectID, teamID string) (IntegrationProjectAccess, error) {
	url := fmt.Sprintf("%s/v1/integrations/configuration/%s/project/%s", c.baseURL, integrationID, projectID)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}

	tflog.Info(ctx, "getting integration project access", map[string]interface{}{
		"url": url,
	})

	type resp struct {
		Allowed bool `json:"allowed"`
	}

	var e resp
	if err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   `{ "allowed": false }`,
	}, &e); err != nil {
		return IntegrationProjectAccess{
			Allowed: false,
			TeamID:  c.teamID(teamID),
		}, err
	}
	return IntegrationProjectAccess{
		Allowed: false,
		TeamID:  c.teamID(teamID),
	}, nil
}
