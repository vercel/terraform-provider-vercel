package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type Network struct {
	AWSAccountID           string    `json:"awsAccountId"`
	AWSAvailabilityZoneIDs *[]string `json:"awsAvailabilityZoneIds"`
	AWSRegion              string    `json:"awsRegion"`
	CIDR                   string    `json:"cidr"`
	CreatedAt              int       `json:"createdAt"`
	EgressIPAddresses      *[]string `json:"egressIpAddresses"`
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	Region                 string    `json:"region"`
	Status                 string    `json:"status"`
	TeamID                 string    `json:"teamId"`
	VPCID                  *string   `json:"vpcId"`
}

type CreateNetworkRequest struct {
	AWSAvailabilityZoneIDs *[]string `json:"awsAvailabilityZoneIds,omitempty"`
	CIDR                   string    `json:"cidr"`
	Name                   string    `json:"name"`
	Region                 string    `json:"region"`
	TeamID                 string    `json:"-"`
}

type CreateNetworkResponse = Network

func (c *Client) CreateNetwork(ctx context.Context, req *CreateNetworkRequest) (r CreateNetworkResponse, err error) {
	url := fmt.Sprintf("%s/v1/connect/networks", c.baseURL)
	if c.TeamID(req.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(req.TeamID))
	}

	body := string(mustMarshal(req))

	tflog.Info(ctx, "Creating Network", map[string]any{
		"body": body,
		"url":  url,
	})

	err = c.doRequest(clientRequest{
		body:   body,
		ctx:    ctx,
		method: "POST",
		url:    url,
	}, &r)

	return r, err
}

type DeleteNetworkRequest struct {
	NetworkID string
	TeamID    string
}

func (c *Client) DeleteNetwork(ctx context.Context, req DeleteNetworkRequest) error {
	url := fmt.Sprintf("%s/v1/connect/networks/%s", c.baseURL, req.NetworkID)
	if c.TeamID(req.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(req.TeamID))
	}

	tflog.Info(ctx, "Deleting Network", map[string]any{"url": url})

	return c.doRequest(clientRequest{
		body:   "",
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}

type ReadNetworkRequest struct {
	NetworkID string
	TeamID    string
}

type ReadNetworkResponse = Network

func (c *Client) ReadNetwork(ctx context.Context, req ReadNetworkRequest) (r ReadNetworkResponse, err error) {
	url := fmt.Sprintf("%s/v1/connect/networks/%s", c.baseURL, req.NetworkID)
	if c.TeamID(req.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(req.TeamID))
	}

	tflog.Info(ctx, "Reading Network", map[string]any{"url": url})

	err = c.doRequest(clientRequest{
		body:   "",
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &r)

	return r, err
}

type UpdateNetworkRequest struct {
	NetworkID string `json:"-"`
	Name      string `json:"name"`
	TeamID    string `json:"-"`
}

type UpdateNetworkResponse = Network

func (c *Client) UpdateNetwork(ctx context.Context, req UpdateNetworkRequest) (r UpdateNetworkResponse, err error) {
	url := fmt.Sprintf("%s/v1/connect/networks/%s", c.baseURL, req.NetworkID)
	if c.TeamID(req.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(req.TeamID))
	}

	body := string(mustMarshal(req))

	tflog.Info(ctx, "Updating Network", map[string]any{
		"body": body,
		"url":  url,
	})

	err = c.doRequest(clientRequest{
		body:   body,
		ctx:    ctx,
		method: "PATCH",
		url:    url,
	}, &r)

	return r, err
}
