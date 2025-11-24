package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type Network struct {
	AWSAccountID           string    `json:"awsAccountId"`
	AWSAvailabilityZoneIds *[]string `json:"awsAvailabilityZoneIds"`
	AWSRegion              string    `json:"awsRegion"`
	Cidr                   string    `json:"cidr"`
	CreatedAt              int       `json:"createdAt"`
	EgressIPAddresses      *[]string `json:"egressIpAddresses"`
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	Status                 string    `json:"status"`
	TeamID                 string    `json:"teamId"`
	VPCID                  *string   `json:"vpcId"`
}

type CreateNetworkRequest struct {
	AWSAvailabilityZoneIds *[]string `json:"awsAvailabilityZoneIds,omitempty"`
	Cidr                   string    `json:"cidr"`
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
