package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// TeamCreateRequest defines the information needed to create a team within vercel.
type TeamCreateRequest struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	Plan string `json:"plan"`
}

type SamlConfig struct {
	Enforced bool              `json:"enforced,omitempty"`
	Roles    map[string]string `json:"roles,omitempty"`
}

type TaxID struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type Address struct {
	Line1      *string `json:"line1"`
	Line2      *string `json:"line2"`
	PostalCode *string `json:"postalCode"`
	City       *string `json:"city"`
	Country    *string `json:"country"`
	State      *string `json:"state"`
}

type RemoteCaching struct {
	Enabled *bool `json:"enabled"`
}

type SpacesConfig struct {
	Enabled bool `json:"enabled"`
}

// Team is the information returned by the vercel api when a team is created.
type Team struct {
	ID                                 string         `json:"id"`
	Name                               string         `json:"name"`
	Avatar                             *string        `json:"avatar"` // hash of uploaded image
	Description                        *string        `json:"description"`
	Slug                               string         `json:"slug"`
	SensitiveEnvironmentVariablePolicy *string        `json:"sensitiveEnvironmentVariablePolicy"`
	EmailDomain                        *string        `json:"emailDomain"`
	Saml                               *SamlConfig    `json:"saml"`
	InviteCode                         *string        `json:"inviteCode"`
	PreviewDeploymentSuffix            *string        `json:"previewDeploymentSuffix"`
	RemoteCaching                      *RemoteCaching `json:"remoteCaching"`
	EnablePreviewFeedback              *string        `json:"enablePreviewFeedback"`
	EnableProductionFeedback           *string        `json:"enableProductionFeedback"`
	Spaces                             *SpacesConfig  `json:"spaces"`
	HideIPAddresses                    *bool          `json:"hideIpAddresses"`
	HideIPAddressesInLogDrains         *bool          `json:"hideIpAddressesInLogDrains,omitempty"`
}

// GetTeam returns information about an existing team within vercel.
func (c *Client) GetTeam(ctx context.Context, idOrSlug string) (t Team, err error) {
	url := fmt.Sprintf("%s/v2/teams/%s", c.baseURL, idOrSlug)
	tflog.Info(ctx, "getting team", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
		body:   "",
	}, &t)
	return t, err
}

type UpdateSamlConfig struct {
	Enforced bool              `json:"enforced"`
	Roles    map[string]string `json:"roles"`
}

type UpdateTeamRequest struct {
	TeamID                             string            `json:"-"`
	Avatar                             string            `json:"avatar,omitempty"`
	Description                        string            `json:"description,omitempty"`
	EmailDomain                        string            `json:"emailDomain,omitempty"`
	Name                               string            `json:"name,omitempty"`
	PreviewDeploymentSuffix            string            `json:"previewDeploymentSuffix,omitempty"`
	Saml                               *UpdateSamlConfig `json:"saml,omitempty"`
	Slug                               string            `json:"slug,omitempty"`
	EnablePreviewFeedback              string            `json:"enablePreviewFeedback,omitempty"`
	EnableProductionFeedback           string            `json:"enableProductionFeedback,omitempty"`
	SensitiveEnvironmentVariablePolicy string            `json:"sensitiveEnvironmentVariablePolicy,omitempty"`
	RemoteCaching                      *RemoteCaching    `json:"remoteCaching,omitempty"`
	HideIPAddresses                    *bool             `json:"hideIpAddresses,omitempty"`
	HideIPAddressesInLogDrains         *bool             `json:"hideIpAddressesInLogDrains,omitempty"`
}

func (c *Client) UpdateTeam(ctx context.Context, request UpdateTeamRequest) (t Team, err error) {
	url := fmt.Sprintf("%s/v2/teams/%s", c.baseURL, request.TeamID)
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating team", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &t)
	return t, err
}
