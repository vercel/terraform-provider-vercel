package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Domain is the information Vercel exposes about an apex domain that has been added to an
// account or team. It is distinct from a ProjectDomain, which associates a domain name with
// a specific project.
type Domain struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	TeamID              string   `json:"-"`
	Verified            bool     `json:"verified"`
	Nameservers         []string `json:"nameservers"`
	IntendedNameservers []string `json:"intendedNameservers"`
	CustomNameservers   []string `json:"customNameservers"`
	Zone                bool     `json:"zone"`
	CreatedAt           *int64   `json:"createdAt"`
	ExpiresAt           *int64   `json:"expiresAt"`
	BoughtAt            *int64   `json:"boughtAt"`
}

// domainResponse unwraps the { "domain": {...} } envelope used by the create and read endpoints.
type domainResponse struct {
	Domain Domain `json:"domain"`
}

// CreateDomainRequest defines the information necessary to add an existing apex domain to Vercel.
type CreateDomainRequest struct {
	Name   string `json:"name"`
	Method string `json:"method"`
	Zone   *bool  `json:"zone,omitempty"`
	TeamID string `json:"-"`
}

// CreateDomain adds an existing apex domain to a Vercel account or team.
func (c *Client) CreateDomain(ctx context.Context, request CreateDomainRequest) (d Domain, err error) {
	url := fmt.Sprintf("%s/v7/domains", c.baseURL)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}

	request.Method = "add"
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating domain", map[string]any{
		"url":     url,
		"payload": payload,
	})
	var response domainResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &response)
	response.Domain.TeamID = c.TeamID(request.TeamID)
	return response.Domain, err
}

// GetDomain retrieves information about an existing apex domain from Vercel.
func (c *Client) GetDomain(ctx context.Context, name, teamID string) (d Domain, err error) {
	url := fmt.Sprintf("%s/v5/domains/%s", c.baseURL, name)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}

	tflog.Info(ctx, "getting domain", map[string]any{
		"url": url,
	})
	var response domainResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &response)
	response.Domain.TeamID = c.TeamID(teamID)
	return response.Domain, err
}

// UpdateDomainRequest defines the information necessary to update an existing apex domain.
// Only the DNS zone is updatable in place; all other changes require replacement.
type UpdateDomainRequest struct {
	Name   string `json:"-"`
	Op     string `json:"op"`
	Zone   *bool  `json:"zone,omitempty"`
	TeamID string `json:"-"`
}

// UpdateDomain updates an existing apex domain within Vercel. The PATCH endpoint returns a
// different shape than the read endpoint, so the domain is re-fetched after updating to
// return a consistent Domain.
func (c *Client) UpdateDomain(ctx context.Context, request UpdateDomainRequest) (d Domain, err error) {
	url := fmt.Sprintf("%s/v3/domains/%s", c.baseURL, request.Name)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}

	request.Op = "update"
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating domain", map[string]any{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, nil)
	if err != nil {
		return d, err
	}

	return c.GetDomain(ctx, request.Name, request.TeamID)
}

// DeleteDomain removes an apex domain from a Vercel account or team.
func (c *Client) DeleteDomain(ctx context.Context, name, teamID string) error {
	url := fmt.Sprintf("%s/v6/domains/%s", c.baseURL, name)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}

	tflog.Info(ctx, "deleting domain", map[string]any{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}
