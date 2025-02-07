package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (c *Client) GetIntegrationProjectAccess(ctx context.Context, integrationID, projectID, teamID string) (bool, error) {
	url := fmt.Sprintf("%s/integrations/configuration/%s/project/%s", c.baseURL, integrationID, projectID)
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
		return false, err
	}
	return e.Allowed, nil
}

func (c *Client) GrantIntegrationProjectAccess(ctx context.Context, integrationID, projectID, teamID string) (bool, error) {
	url := fmt.Sprintf("%s/integrations/configuration/%s/project/%s", c.baseURL, integrationID, projectID)
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
		return false, err
	}
	return true, nil
}

func (c *Client) RevokeIntegrationProjectAccess(ctx context.Context, integrationID, projectID, teamID string) (bool, error) {
	url := fmt.Sprintf("%s/integrations/configuration/%s/project/%s", c.baseURL, integrationID, projectID)
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
		return false, err
	}
	return false, nil
}
