package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// KMSIssuer represents a Vercel KMS issuer. The KMS CRUD API lives under
// `/v1/kms/issuers` (see the api-kms service) and is gated behind a feature
// flag; when the flag is disabled the API intentionally returns 404.
type KMSIssuer struct {
	ID          string            `json:"id"`
	OwnerID     string            `json:"ownerId"`
	Name        string            `json:"name"`
	Algorithm   string            `json:"algorithm"`
	Origin      string            `json:"origin"`
	ManagedBy   string            `json:"managedBy,omitempty"`
	CreatedAt   string            `json:"createdAt"`
	UpdatedAt   string            `json:"updatedAt"`
	SigningKeys []KMSSigningKey   `json:"signingKeys"`
	Policies    []KMSIssuerPolicy `json:"policies"`
	// TeamID is threaded through the query string rather than the request body.
	TeamID string `json:"-"`
}

// KMSSigningKey is a signing key belonging to an issuer. Private key material is
// never returned by the API. `publicKey` is a free-form JWK object.
type KMSSigningKey struct {
	KeyID                string          `json:"keyId"`
	IssuerID             string          `json:"issuerId"`
	Algorithm            string          `json:"algorithm"`
	Status               string          `json:"status"`
	PublicKey            json.RawMessage `json:"publicKey,omitempty"`
	PublicKeyFingerprint string          `json:"publicKeyFingerprint,omitempty"`
	CreatedAt            string          `json:"createdAt"`
	UpdatedAt            string          `json:"updatedAt"`
	RevokeAt             string          `json:"revokeAt,omitempty"`
}

// KMSIssuerPolicy is an authorization policy attached to an issuer. Only
// project-grant policies are managed by this provider; `clientId` is retained
// solely so that any connex-grant policies present on an issuer can be decoded
// without error.
type KMSIssuerPolicy struct {
	Kind         string          `json:"kind"`
	TeamID       string          `json:"teamId,omitempty"`
	ProjectID    string          `json:"projectId,omitempty"`
	ClientID     string          `json:"clientId,omitempty"`
	Environments []string        `json:"environments,omitempty"`
	TokenClaims  json.RawMessage `json:"tokenClaims,omitempty"`
	CreatedAt    string          `json:"createdAt"`
	UpdatedAt    string          `json:"updatedAt"`
}

func (c *Client) kmsIssuersURL(teamID string) string {
	url := fmt.Sprintf("%s/v1/kms/issuers", c.baseURL)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	return url
}

func (c *Client) kmsIssuerURL(issuerID, teamID string) string {
	url := fmt.Sprintf("%s/v1/kms/issuers/%s", c.baseURL, issuerID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	return url
}

type CreateKMSIssuerRequest struct {
	Name      string `json:"name"`
	Algorithm string `json:"algorithm,omitempty"`
	ImportKey string `json:"importKey,omitempty"`
	KeyID     string `json:"keyId,omitempty"`
	TeamID    string `json:"-"`
}

func (c *Client) CreateKMSIssuer(ctx context.Context, request CreateKMSIssuerRequest) (i KMSIssuer, err error) {
	url := c.kmsIssuersURL(request.TeamID)
	body := string(mustMarshal(request))
	tflog.Info(ctx, "creating kms issuer", map[string]any{
		"url":  url,
		"body": body,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   body,
	}, &i)
	if err != nil {
		return i, err
	}
	i.TeamID = c.TeamID(request.TeamID)
	return i, nil
}

func (c *Client) GetKMSIssuer(ctx context.Context, issuerID, teamID string) (i KMSIssuer, err error) {
	url := c.kmsIssuerURL(issuerID, teamID)
	tflog.Info(ctx, "reading kms issuer", map[string]any{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &i)
	if err != nil {
		return i, err
	}
	i.TeamID = c.TeamID(teamID)
	return i, nil
}

type kmsIssuersResponse struct {
	Issuers    []KMSIssuer `json:"issuers"`
	Pagination struct {
		Count int    `json:"count"`
		Next  string `json:"next"`
	} `json:"pagination"`
}

// ListKMSIssuers returns the issuers for a team. The list endpoint omits nested
// `signingKeys`/`policies`; use GetKMSIssuer to hydrate a single issuer.
func (c *Client) ListKMSIssuers(ctx context.Context, teamID string) (issuers []KMSIssuer, err error) {
	url := c.kmsIssuersURL(teamID)
	tflog.Info(ctx, "listing kms issuers", map[string]any{
		"url": url,
	})
	var response kmsIssuersResponse
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "GET",
		url:    url,
	}, &response)
	if err != nil {
		return nil, err
	}
	for i := range response.Issuers {
		response.Issuers[i].TeamID = c.TeamID(teamID)
	}
	return response.Issuers, nil
}

type UpdateKMSIssuerRequest struct {
	Name     string `json:"name"`
	IssuerID string `json:"-"`
	TeamID   string `json:"-"`
}

func (c *Client) UpdateKMSIssuer(ctx context.Context, request UpdateKMSIssuerRequest) (i KMSIssuer, err error) {
	url := c.kmsIssuerURL(request.IssuerID, request.TeamID)
	body := string(mustMarshal(struct {
		Name string `json:"name"`
	}{Name: request.Name}))
	tflog.Info(ctx, "updating kms issuer", map[string]any{
		"url":  url,
		"body": body,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   body,
	}, &i)
	if err != nil {
		return i, err
	}
	i.TeamID = c.TeamID(request.TeamID)
	return i, nil
}

func (c *Client) DeleteKMSIssuer(ctx context.Context, issuerID, teamID string) error {
	url := c.kmsIssuerURL(issuerID, teamID)
	tflog.Info(ctx, "deleting kms issuer", map[string]any{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}

type RotateKMSIssuerKeyRequest struct {
	RevokePreviousAt string `json:"revokePreviousAt,omitempty"`
	ImportKey        string `json:"importKey,omitempty"`
	KeyID            string `json:"keyId,omitempty"`
	IssuerID         string `json:"-"`
	TeamID           string `json:"-"`
}

func (c *Client) RotateKMSIssuerKey(ctx context.Context, request RotateKMSIssuerKeyRequest) (k KMSSigningKey, err error) {
	url := fmt.Sprintf("%s/v1/kms/issuers/%s/keys/rotate", c.baseURL, request.IssuerID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	body := string(mustMarshal(request))
	tflog.Info(ctx, "rotating kms issuer key", map[string]any{
		"url":  url,
		"body": body,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   body,
	}, &k)
	if err != nil {
		return k, err
	}
	return k, nil
}

type CreateKMSProjectGrantRequest struct {
	IssuerID     string          `json:"-"`
	TeamID       string          `json:"-"`
	ProjectID    string          `json:"-"`
	Environments []string        `json:"-"`
	TokenClaims  json.RawMessage `json:"-"`
}

func (c *Client) CreateKMSProjectGrant(ctx context.Context, request CreateKMSProjectGrantRequest) (p KMSIssuerPolicy, err error) {
	url := fmt.Sprintf("%s/v1/kms/issuers/%s/policies", c.baseURL, request.IssuerID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	body := string(mustMarshal(struct {
		Kind         string          `json:"kind"`
		ProjectID    string          `json:"projectId"`
		Environments []string        `json:"environments"`
		TokenClaims  json.RawMessage `json:"tokenClaims,omitempty"`
	}{
		Kind:         "project-grant",
		ProjectID:    request.ProjectID,
		Environments: request.Environments,
		TokenClaims:  request.TokenClaims,
	}))
	tflog.Info(ctx, "creating kms project grant", map[string]any{
		"url":  url,
		"body": body,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "POST",
		url:    url,
		body:   body,
	}, &p)
	if err != nil {
		return p, err
	}
	return p, nil
}

type UpdateKMSProjectGrantRequest struct {
	IssuerID     string          `json:"-"`
	TeamID       string          `json:"-"`
	ProjectID    string          `json:"-"`
	Environments []string        `json:"environments,omitempty"`
	TokenClaims  json.RawMessage `json:"tokenClaims,omitempty"`
}

func (c *Client) UpdateKMSProjectGrant(ctx context.Context, request UpdateKMSProjectGrantRequest) (p KMSIssuerPolicy, err error) {
	url := fmt.Sprintf("%s/v1/kms/issuers/%s/policies/project-grant/%s", c.baseURL, request.IssuerID, request.ProjectID)
	if c.TeamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(request.TeamID))
	}
	body := string(mustMarshal(struct {
		Environments []string        `json:"environments,omitempty"`
		TokenClaims  json.RawMessage `json:"tokenClaims,omitempty"`
	}{
		Environments: request.Environments,
		TokenClaims:  request.TokenClaims,
	}))
	tflog.Info(ctx, "updating kms project grant", map[string]any{
		"url":  url,
		"body": body,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: "PATCH",
		url:    url,
		body:   body,
	}, &p)
	if err != nil {
		return p, err
	}
	return p, nil
}

func (c *Client) DeleteKMSProjectGrant(ctx context.Context, issuerID, projectID, teamID string) error {
	url := fmt.Sprintf("%s/v1/kms/issuers/%s/policies/project-grant/%s", c.baseURL, issuerID, projectID)
	if c.TeamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.TeamID(teamID))
	}
	tflog.Info(ctx, "deleting kms project grant", map[string]any{
		"url": url,
	})
	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: "DELETE",
		url:    url,
	}, nil)
}
