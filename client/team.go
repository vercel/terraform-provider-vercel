package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// TeamCreateRequest defines the information needed to create a team within vercel.
type TeamCreateRequest struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	Plan string `json:"plan"`
}

type SamlRoleAccessGroupID struct {
	AccessGroupID string `json:"accessGroupId"`
}

type SamlRoleAPI struct {
	Role          *string
	AccessGroupID *SamlRoleAccessGroupID
}

type SamlRolesAPI map[string]SamlRoleAPI

type SamlRole struct {
	Role          *string `json:"role"`
	AccessGroupID *string `json:"access_group_id"`
}

type SamlRoles map[string]SamlRole

func (f *SamlRoleAPI) UnmarshalJSON(data []byte) error {
	var role string
	if err := json.Unmarshal(data, &role); err == nil {
		f.Role = &role
		return nil
	}
	var ag SamlRoleAccessGroupID
	if err := json.Unmarshal(data, &ag); err == nil {
		f.AccessGroupID = &ag
		return nil
	}
	return fmt.Errorf("received json is neither Role string nor AccessGroupID map")
}

func (f *SamlRoles) UnmarshalJSON(data []byte) error {
	var result SamlRolesAPI
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	tmp := make(SamlRoles)
	for k, v := range result {
		k := k
		v := v
		if v.Role != nil {
			tmp[k] = SamlRole{
				Role: v.Role,
			}
		}
		if v.AccessGroupID != nil {
			tmp[k] = SamlRole{
				AccessGroupID: &v.AccessGroupID.AccessGroupID,
			}
		}
	}
	*f = tmp
	return nil
}

type SamlConfig struct {
	Enforced bool      `json:"enforced,omitempty"`
	Roles    SamlRoles `json:"roles,omitempty"`
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
	tflog.Info(ctx, "getting team", map[string]any{
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
	Enforced bool           `json:"enforced"`
	Roles    map[string]any `json:"roles"`
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
	tflog.Info(ctx, "updating team", map[string]any{
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
