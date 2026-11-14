package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// OrganizationTeam describes a team's membership of an organization.
type OrganizationTeam struct {
	OrganizationID string `json:"organizationId"`
	TeamID         string `json:"teamId"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	BillingPlan    string `json:"billingPlan"`
}

// OrganizationTeamsPage contains one page of an organization's teams.
type OrganizationTeamsPage struct {
	Teams  []OrganizationTeam `json:"teams"`
	Cursor string             `json:"cursor"`
}

// CreateOrganizationTeam creates a child team using the organization's billing setup.
func (c *Client) CreateOrganizationTeam(ctx context.Context, organizationID, name, slug, billingPlan string) (OrganizationTeam, error) {
	var team OrganizationTeam
	err := c.doRequest(clientRequest{
		ctx: ctx, method: http.MethodPost,
		url:  fmt.Sprintf("%s/v1/organizations/%s/teams", c.baseURL, url.PathEscape(organizationID)),
		body: string(mustMarshal(map[string]string{"mode": "create", "name": name, "slug": slug, "billingPlan": billingPlan})),
	}, &team)
	return team, err
}

// ListOrganizationTeams lists one page of teams from the organization's eventually consistent index.
func (c *Client) ListOrganizationTeams(ctx context.Context, organizationID, cursor string) (OrganizationTeamsPage, error) {
	endpoint := fmt.Sprintf("%s/v1/organizations/%s/teams", c.baseURL, url.PathEscape(organizationID))
	if cursor != "" {
		endpoint += "?cursor=" + url.QueryEscape(cursor)
	}
	var page OrganizationTeamsPage
	err := c.doRequest(clientRequest{ctx: ctx, method: http.MethodGet, url: endpoint}, &page)
	return page, err
}

// GetOrganizationTeam reads a team and its billing profile from the organization.
func (c *Client) GetOrganizationTeam(ctx context.Context, organizationID, teamID string) (OrganizationTeam, error) {
	cursor := ""
	for {
		page, err := c.ListOrganizationTeams(ctx, organizationID, cursor)
		if err != nil {
			return OrganizationTeam{}, err
		}
		for _, team := range page.Teams {
			if team.TeamID == teamID {
				// Legacy organization links omit the billing profile and use platform billing.
				if team.BillingPlan == "" {
					team.BillingPlan = "platform"
				}
				return team, nil
			}
		}
		if page.Cursor == "" {
			break
		}
		if page.Cursor == cursor {
			return OrganizationTeam{}, fmt.Errorf("organization teams API returned a repeated cursor")
		}
		cursor = page.Cursor
	}
	// List absence is not proof of deletion because the index is eventually
	// consistent. Confirm the team's current organization with a point read.
	var organization struct {
		ID *string `json:"organizationId"`
	}
	if err := c.doRequest(clientRequest{
		ctx: ctx, method: http.MethodGet,
		url: fmt.Sprintf("%s/v1/teams/%s/organization", c.baseURL, url.PathEscape(teamID)),
	}, &organization); err != nil {
		return OrganizationTeam{}, fmt.Errorf("could not confirm team's organization: %w", err)
	}
	if organization.ID != nil && *organization.ID == organizationID {
		return OrganizationTeam{}, APIError{
			StatusCode: http.StatusServiceUnavailable,
			Code:       "organization_team_not_yet_visible",
			Message:    "Team is still linked to the organization but is missing from its teams list. Retry the operation.",
		}
	}
	return OrganizationTeam{}, APIError{StatusCode: http.StatusNotFound, Code: "organization_team_not_found", Message: "Team not found in organization"}
}

// DeleteOrganizationTeam deletes the team, rather than detaching it from the organization.
func (c *Client) DeleteOrganizationTeam(ctx context.Context, organizationID, teamID string) error {
	return c.doRequest(clientRequest{
		ctx: ctx, method: http.MethodDelete,
		url: fmt.Sprintf("%s/v1/organizations/%s/teams/%s", c.baseURL, url.PathEscape(organizationID), url.PathEscape(teamID)),
	}, nil)
}

// GetOrganizationRootTeamID returns the billing root team, which cannot be managed as a child.
func (c *Client) GetOrganizationRootTeamID(ctx context.Context, organizationID string) (string, error) {
	var organization struct {
		TeamID string `json:"teamId"`
	}
	err := c.doRequest(clientRequest{ctx: ctx, method: http.MethodGet, url: fmt.Sprintf("%s/v1/organizations/%s", c.baseURL, url.PathEscape(organizationID))}, &organization)
	return organization.TeamID, err
}
