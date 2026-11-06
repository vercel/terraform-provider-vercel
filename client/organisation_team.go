package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// OrganisationTeam describes a team's membership of an organisation.
type OrganisationTeam struct {
	OrganisationID string `json:"organizationId"`
	TeamID         string `json:"teamId"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	BillingPlan    string `json:"billingPlan"`
}

// CreateOrganisationTeam creates a child team using the organisation's billing setup.
func (c *Client) CreateOrganisationTeam(ctx context.Context, organisationID, name, slug, billingPlan string) (OrganisationTeam, error) {
	var team OrganisationTeam
	err := c.doRequest(clientRequest{
		ctx: ctx, method: http.MethodPost,
		url:  fmt.Sprintf("%s/v1/organizations/%s/teams", c.baseURL, url.PathEscape(organisationID)),
		body: string(mustMarshal(map[string]string{"mode": "create", "name": name, "slug": slug, "billingPlan": billingPlan})),
	}, &team)
	return team, err
}

// GetOrganisationTeam reads a team and its billing profile from the organisation.
func (c *Client) GetOrganisationTeam(ctx context.Context, organisationID, teamID string) (OrganisationTeam, error) {
	cursor := ""
	for {
		endpoint := fmt.Sprintf("%s/v1/organizations/%s/teams", c.baseURL, url.PathEscape(organisationID))
		if cursor != "" {
			endpoint += "?cursor=" + url.QueryEscape(cursor)
		}
		var page struct {
			Teams  []OrganisationTeam `json:"teams"`
			Cursor string             `json:"cursor"`
		}
		if err := c.doRequest(clientRequest{ctx: ctx, method: http.MethodGet, url: endpoint}, &page); err != nil {
			return OrganisationTeam{}, err
		}
		for _, team := range page.Teams {
			if team.TeamID == teamID {
				// Legacy organisation links omit the billing profile and use platform billing.
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
			return OrganisationTeam{}, fmt.Errorf("organisation teams API returned a repeated cursor")
		}
		cursor = page.Cursor
	}
	return OrganisationTeam{}, APIError{StatusCode: http.StatusNotFound, Code: "organisation_team_not_found", Message: "Team not found in organisation"}
}

// DeleteOrganisationTeam deletes the team, rather than detaching it from the organisation.
func (c *Client) DeleteOrganisationTeam(ctx context.Context, organisationID, teamID string) error {
	return c.doRequest(clientRequest{
		ctx: ctx, method: http.MethodDelete,
		url: fmt.Sprintf("%s/v1/organizations/%s/teams/%s", c.baseURL, url.PathEscape(organisationID), url.PathEscape(teamID)),
	}, nil)
}

// GetOrganisationRootTeamID returns the billing root team, which cannot be managed as a child.
func (c *Client) GetOrganisationRootTeamID(ctx context.Context, organisationID string) (string, error) {
	var organisation struct {
		TeamID string `json:"teamId"`
	}
	err := c.doRequest(clientRequest{ctx: ctx, method: http.MethodGet, url: fmt.Sprintf("%s/v1/organizations/%s", c.baseURL, url.PathEscape(organisationID))}, &organisation)
	return organisation.TeamID, err
}
