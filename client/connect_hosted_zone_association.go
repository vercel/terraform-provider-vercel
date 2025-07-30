package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type HostedZoneAssociation struct {
	HostedZoneID   string `json:"hostedZoneId"`
	HostedZoneName string `json:"hostedZoneName"`
	Owner          string `json:"owner"`
}

type GetHostedZoneAssociationRequest struct {
	ConfigurationID string
	HostedZoneID    string
	TeamID          string
}

func (c *Client) GetHostedZoneAssociation(ctx context.Context, req GetHostedZoneAssociationRequest) (r HostedZoneAssociation, err error) {
	url := fmt.Sprintf("%s/v1/connect/configurations/%s/hosted-zones/%s", c.baseURL, req.ConfigurationID, req.HostedZoneID)
	if c.TeamID(req.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(req.TeamID))
	}

	tflog.Info(ctx, "Getting Hosted Zone Association", map[string]any{"url": url})

	err = c.doRequest(clientRequest{
		body:   "",
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &r)

	return r, err
}

type CreateHostedZoneAssociationRequest struct {
	ConfigurationID string
	HostedZoneID    string
	TeamID          string
}

func (c *Client) CreateHostedZoneAssociation(ctx context.Context, req CreateHostedZoneAssociationRequest) (r HostedZoneAssociation, err error) {
	url := fmt.Sprintf("%s/v1/connect/configurations/%s/hosted-zones", c.baseURL, req.ConfigurationID)
	if c.TeamID(req.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(req.TeamID))
	}

	body := string(mustMarshal(
		struct {
			HostedZoneId string `json:"hostedZoneId"`
		}{
			HostedZoneId: req.HostedZoneID,
		},
	))

	tflog.Info(ctx, "Creating Hosted Zone Association", map[string]any{
		"body": body,
		"url":  url,
	})

	err = c.doRequest(clientRequest{ // TODO: This endpoint actually returns a different shape, we should change it to return HostedZoneAssociation
		body:   body,
		ctx:    ctx,
		method: "POST",
		url:    url,
	}, &r)

	return r, err
}

type DeleteHostedZoneAssociationRequest struct {
	ConfigurationID string
	HostedZoneID    string
	TeamID          string
}

func (c *Client) DeleteHostedZoneAssociation(ctx context.Context, req DeleteHostedZoneAssociationRequest) error {
	url := fmt.Sprintf("%s/v1/connect/configurations/%s/hosted-zones/%s", c.baseURL, req.ConfigurationID, req.HostedZoneID)
	if c.TeamID(req.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(req.TeamID))
	}

	tflog.Info(ctx, "Deleting Hosted Zone Association", map[string]any{"url": url})

	return c.doRequest(clientRequest{
		body:   "",
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}
