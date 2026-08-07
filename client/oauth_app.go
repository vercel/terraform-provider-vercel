package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// OAuthAppNotFound detects the error returned when an OAuth app does not
// exist. Unusually, the get endpoint reports a missing app as HTTP 400 with
// code "invalid_client" (the OAuth-style error), while delete uses a plain
// 404 — treat both as not-found.
func OAuthAppNotFound(err error) bool {
	var apiErr APIError
	return err != nil && errors.As(err, &apiErr) && (apiErr.StatusCode == 404 || apiErr.Code == "invalid_client")
}

// OAuthAppClientSecretMetadata contains the non-sensitive metadata the API
// exposes about an OAuth app's client secrets. The secret value itself is only
// ever returned once, by CreateOAuthAppSecret.
type OAuthAppClientSecretMetadata struct {
	ID            string `json:"id"`
	LastFourChars string `json:"lastFourChars"`
}

// OAuthApp represents a "Sign in with Vercel" OAuth application.
type OAuthApp struct {
	ClientID          string                         `json:"clientId"`
	TeamID            string                         `json:"teamId"`
	Name              string                         `json:"name"`
	Slug              string                         `json:"slug"`
	Description       string                         `json:"description"`
	HomePageURI       string                         `json:"homePageUri"`
	RedirectURIs      []string                       `json:"redirectUris"`
	Scopes            []string                       `json:"scopes"`
	Permissions       []string                       `json:"permissions"`
	PrivacyPolicyURL  string                         `json:"privacyPolicyUrl"`
	TermsOfServiceURL string                         `json:"termsOfServiceUrl"`
	CodeOfConductURL  string                         `json:"codeOfConductUrl"`
	ClientSecrets     []OAuthAppClientSecretMetadata `json:"clientSecrets"`
}

type CreateOAuthAppRequest struct {
	TeamID            string   `json:"-"`
	Name              string   `json:"name"`
	Slug              string   `json:"slug"`
	Description       string   `json:"description,omitempty"`
	HomePageURI       string   `json:"homePageUri,omitempty"`
	RedirectURIs      []string `json:"redirectUris,omitempty"`
	Scopes            []string `json:"scopes,omitempty"`
	PrivacyPolicyURL  string   `json:"privacyPolicyUrl,omitempty"`
	TermsOfServiceURL string   `json:"termsOfServiceUrl,omitempty"`
	CodeOfConductURL  string   `json:"codeOfConductUrl,omitempty"`
}

// UpdateOAuthAppRequest updates an OAuth app. Nullable URL fields are pointers
// WITHOUT omitempty: an explicit JSON null is how the API clears a previously
// set value, so unset (nil) pointers are serialized as null deliberately.
type UpdateOAuthAppRequest struct {
	TeamID            string   `json:"-"`
	ClientID          string   `json:"-"`
	Name              string   `json:"name"`
	Slug              string   `json:"slug"`
	Description       string   `json:"description"`
	HomePageURI       *string  `json:"homePageUri"`
	RedirectURIs      []string `json:"redirectUris"`
	Scopes            []string `json:"scopes"`
	PrivacyPolicyURL  *string  `json:"privacyPolicyUrl"`
	TermsOfServiceURL *string  `json:"termsOfServiceUrl"`
	CodeOfConductURL  *string  `json:"codeOfConductUrl"`
}

// OAuthAppSecret is the response of generating a client secret. This is the
// ONLY time the API returns the plaintext secret; subsequent reads expose just
// its last four characters.
type OAuthAppSecret struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

func (c *Client) CreateOAuthApp(ctx context.Context, request CreateOAuthAppRequest) (a OAuthApp, err error) {
	url := fmt.Sprintf("%s/v1/oauth-apps", c.baseURL)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating oauth app", map[string]any{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   payload,
	}, &a)
	return a, err
}

func (c *Client) GetOAuthApp(ctx context.Context, clientID, teamID string) (OAuthApp, error) {
	url := fmt.Sprintf("%s/v1/oauth-apps/%s", c.baseURL, clientID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "getting oauth app", map[string]any{
		"url": url,
	})
	// Unlike create/update, the get endpoint wraps the app in an envelope.
	var response struct {
		App OAuthApp `json:"app"`
	}
	err := c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &response)
	return response.App, err
}

func (c *Client) UpdateOAuthApp(ctx context.Context, request UpdateOAuthAppRequest) (a OAuthApp, err error) {
	// The request intentionally omits `omitempty` so explicit nulls can clear
	// nullable URL fields — but the array fields must never serialize as null
	// (the API expects arrays; an empty array is how "none" is expressed).
	// Normalize nil slices so no caller can send `"redirectUris": null` etc.
	if request.RedirectURIs == nil {
		request.RedirectURIs = []string{}
	}
	if request.Scopes == nil {
		request.Scopes = []string{}
	}
	url := fmt.Sprintf("%s/v1/oauth-apps/%s", c.baseURL, request.ClientID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "updating oauth app", map[string]any{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &a)
	return a, err
}

// UpdateOAuthAppPermissions sets the Vercel REST API permissions granted to
// an OAuth app's tokens. This is the ONLY call that writes the permissions
// field — the general UpdateOAuthApp deliberately omits it, so the two can be
// managed independently without clobbering each other.
func (c *Client) UpdateOAuthAppPermissions(ctx context.Context, clientID string, permissions []string, teamID string) (a OAuthApp, err error) {
	url := fmt.Sprintf("%s/v1/oauth-apps/%s", c.baseURL, clientID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	if permissions == nil {
		permissions = []string{}
	}
	payload := string(mustMarshal(struct {
		Permissions []string `json:"permissions"`
	}{Permissions: permissions}))
	tflog.Info(ctx, "updating oauth app permissions", map[string]any{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   payload,
	}, &a)
	return a, err
}

func (c *Client) DeleteOAuthApp(ctx context.Context, clientID, teamID string) error {
	url := fmt.Sprintf("%s/v1/oauth-apps/%s", c.baseURL, clientID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "deleting oauth app", map[string]any{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}

func (c *Client) CreateOAuthAppSecret(ctx context.Context, clientID, teamID string) (s OAuthAppSecret, err error) {
	url := fmt.Sprintf("%s/v1/oauth-apps/%s/secret", c.baseURL, clientID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "creating oauth app client secret", map[string]any{
		"url": url,
	})
	// The endpoint takes no parameters but rejects body-less requests with
	// 415 Unsupported Media Type — send an empty JSON object.
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   "{}",
	}, &s)
	return s, err
}

// DeleteOAuthAppSecret deletes a client secret. The API identifies secrets by
// the LAST FOUR CHARACTERS of the secret value, not by id.
func (c *Client) DeleteOAuthAppSecret(ctx context.Context, clientID, lastFourChars, teamID string) error {
	url := fmt.Sprintf("%s/v1/oauth-apps/%s/secret/%s", c.baseURL, clientID, lastFourChars)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "deleting oauth app client secret", map[string]any{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}
