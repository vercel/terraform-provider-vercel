package vercel

import (
	"fmt"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

// ShareableLink contains the state common to the resource and data source.
type ShareableLink struct {
	ID        types.String `tfsdk:"id"`
	Alias     types.String `tfsdk:"alias"`
	TeamID    types.String `tfsdk:"team_id"`
	ProjectID types.String `tfsdk:"project_id"`
	Secret    types.String `tfsdk:"secret"`
	URL       types.String `tfsdk:"url"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	CreatedBy types.String `tfsdk:"created_by"`
	ExpiresAt types.Int64  `tfsdk:"expires_at"`
	Active    types.Bool   `tfsdk:"active"`
}

func findShareableLink(bypasses map[string]client.ProtectionBypass) (string, client.ProtectionBypass, error) {
	var secret string
	var link client.ProtectionBypass
	for key, bypass := range bypasses {
		if bypass.Scope != "shareable-link" {
			continue
		}
		if secret != "" {
			return "", client.ProtectionBypass{}, fmt.Errorf("alias contains multiple shareable links")
		}
		secret, link = key, bypass
	}
	if secret == "" {
		return "", client.ProtectionBypass{}, fmt.Errorf("alias has no readable shareable link; check that a link exists and the API token has permission to read protection bypasses")
	}
	return secret, link, nil
}

func shareableLinkState(alias client.AliasResponse, secret string, link client.ProtectionBypass) ShareableLink {
	return ShareableLink{
		ID: types.StringValue(alias.UID), Alias: types.StringValue(alias.Alias), TeamID: toTeamID(alias.TeamID),
		ProjectID: types.StringValue(alias.ProjectID), Secret: types.StringValue(secret),
		URL:       types.StringValue("https://" + alias.Alias + "/?_vercel_share=" + url.QueryEscape(secret)),
		CreatedAt: types.Int64Value(link.CreatedAt), CreatedBy: types.StringValue(link.CreatedBy),
		ExpiresAt: types.Int64PointerValue(link.Expires), Active: types.BoolValue(link.Expires == nil || *link.Expires > time.Now().Unix()),
	}
}
