package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const oidcFederationPoliciesPath = "/v1/oauth-apps/oidc-federation/policies"

type OIDCClaimValue struct {
	Value     string `json:"value"`
	Wildcards bool   `json:"wildcards"`
}

type OIDCClaim struct {
	Name   string           `json:"name"`
	Values []OIDCClaimValue `json:"values"`
}

type OIDCResources struct {
	ProjectIDs []string `json:"projectIds"`
}

type OIDCFederationPolicy struct {
	PolicyID    string         `json:"policyId"`
	ClientID    string         `json:"clientId"`
	IssuerURL   string         `json:"issuerUrl"`
	TeamID      string         `json:"teamId"`
	Name        *string        `json:"name"`
	Claims      []OIDCClaim    `json:"claims"`
	Permissions []string       `json:"permissions"`
	Commands    []string       `json:"commands"`
	Resources   *OIDCResources `json:"resources"`
}

type CreateOIDCFederationPolicyRequest struct {
	TeamID      string         `json:"-"`
	ClientID    string         `json:"clientId"`
	IssuerURL   string         `json:"issuerUrl"`
	Name        *string        `json:"name,omitempty"`
	Claims      []OIDCClaim    `json:"claims"`
	Permissions []string       `json:"permissions,omitempty"`
	Commands    []string       `json:"commands,omitempty"`
	Resources   *OIDCResources `json:"resources,omitempty"`
}

type UpdateOIDCFederationPolicyRequest struct {
	TeamID      string         `json:"-"`
	PolicyID    string         `json:"-"`
	Name        *string        `json:"name,omitempty"`
	Claims      *[]OIDCClaim   `json:"claims,omitempty"`
	Permissions *[]string      `json:"permissions,omitempty"`
	Commands    *[]string      `json:"commands,omitempty"`
	Resources   *OIDCResources `json:"resources,omitempty"`
}

func (c *Client) oidcFederationPoliciesURL(teamID string) string {
	query := url.Values{}
	if id := c.TeamID(teamID); id != "" {
		query.Set("teamId", id)
	}

	endpoint := fmt.Sprintf("%s%s", c.baseURL, oidcFederationPoliciesPath)
	if encoded := query.Encode(); encoded != "" {
		endpoint = fmt.Sprintf("%s?%s", endpoint, encoded)
	}
	return endpoint
}

func (c *Client) oidcFederationPolicyURL(teamID, policyID string) string {
	query := url.Values{}
	if id := c.TeamID(teamID); id != "" {
		query.Set("teamId", id)
	}

	endpoint := fmt.Sprintf("%s%s/%s", c.baseURL, oidcFederationPoliciesPath, url.PathEscape(policyID))
	if encoded := query.Encode(); encoded != "" {
		endpoint = fmt.Sprintf("%s?%s", endpoint, encoded)
	}
	return endpoint
}

func (c *Client) CreateOIDCFederationPolicy(ctx context.Context, request CreateOIDCFederationPolicyRequest) (policy OIDCFederationPolicy, err error) {
	endpoint := c.oidcFederationPoliciesURL(request.TeamID)
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating OIDC federation policy", map[string]any{
		"url":     endpoint,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    endpoint,
		body:   payload,
	}, &policy)
	return policy, err
}

func (c *Client) GetOIDCFederationPolicy(ctx context.Context, policyID, teamID string) (OIDCFederationPolicy, error) {
	endpoint := c.oidcFederationPoliciesURL(teamID)
	tflog.Info(ctx, "getting OIDC federation policy", map[string]any{
		"url":       endpoint,
		"policy_id": policyID,
	})
	var response struct {
		Policies []OIDCFederationPolicy `json:"policies"`
	}
	if err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    endpoint,
	}, &response); err != nil {
		return OIDCFederationPolicy{}, err
	}
	for _, policy := range response.Policies {
		if policy.PolicyID == policyID {
			return policy, nil
		}
	}
	return OIDCFederationPolicy{}, APIError{
		Code:       "not_found",
		Message:    "OIDC federation policy not found",
		StatusCode: 404,
	}
}

func (c *Client) UpdateOIDCFederationPolicy(ctx context.Context, request UpdateOIDCFederationPolicyRequest) (policy OIDCFederationPolicy, err error) {
	endpoint := c.oidcFederationPolicyURL(request.TeamID, request.PolicyID)
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating OIDC federation policy", map[string]any{
		"url":     endpoint,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    endpoint,
		body:   payload,
	}, &policy)
	return policy, err
}

func (c *Client) DeleteOIDCFederationPolicy(ctx context.Context, policyID, teamID string) error {
	endpoint := c.oidcFederationPolicyURL(teamID, policyID)
	tflog.Info(ctx, "deleting OIDC federation policy", map[string]any{
		"url": endpoint,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    endpoint,
	}, nil)
}
