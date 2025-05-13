package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// SecureComputeNetworkAWS represents the AWS configuration part of a Secure Compute Network.
type SecureComputeNetworkAWS struct {
	AccountID          string   `json:"AccountId"`
	Region             string   `json:"Region"`
	ElasticIpAddresses []string `json:"ElasticIpAddresses,omitempty"`
	LambdaRoleArn      *string  `json:"LambdaRoleArn,omitempty"`
	SecurityGroupId    *string  `json:"SecurityGroupId,omitempty"`
	StackId            *string  `json:"StackId,omitempty"`
	SubnetIds          []string `json:"SubnetIds,omitempty"`
	SubscriptionArn    *string  `json:"SubscriptionArn,omitempty"`
	VpcId              *string  `json:"VpcId,omitempty"`
}

// SecureComputeNetwork represents a Vercel Secure Compute Network configuration.
type SecureComputeNetwork struct {
	DC                      string   `json:"dc"`
	ProjectIDs              []string `json:"projectIds,omitempty"`
	ProjectsCount           *int     `json:"projectsCount,omitempty"`
	PeeringConnectionsCount *int     `json:"peeringConnectionsCount,omitempty"`

	AWS                 SecureComputeNetworkAWS `json:"AWS"`
	ConfigurationName   string                  `json:"ConfigurationName"`
	ID                  string                  `json:"Id"`
	TeamID              string                  `json:"TeamId"`
	CIDRBlock           *string                 `json:"CidrBlock,omitempty"`
	AvailabilityZoneIDs []string                `json:"AvailabilityZoneIds,omitempty"`
	Version             string                  `json:"Version"`
	ConfigurationStatus string                  `json:"ConfigurationStatus"`
}

// ListSecureComputeNetworks fetches all Secure Compute Networks for a team.
func (c *Client) ListSecureComputeNetworks(
	ctx context.Context,
	teamID string,
) ([]SecureComputeNetwork, error) {
	url := fmt.Sprintf("%s/v1/connect/configurations", c.baseURL)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "reading secure compute networks", map[string]any{
		"url":     url,
		"team_id": c.TeamID(teamID),
	})

	var networks []SecureComputeNetwork
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &networks)

	return networks, err
}
