package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	oidcTokenExchangePath = "/login/oauth/token"
	// oidcTokenRefreshLeeway is how long before it expires an access token is
	// exchanged again, so requests are not sent with a token about to expire.
	oidcTokenRefreshLeeway = time.Minute
	// oidcTokenRefreshRetryInterval is how long to wait after a failed exchange
	// before trying again, while the current access token is still valid.
	oidcTokenRefreshRetryInterval = 10 * time.Second
)

// errOIDCTokenExchange marks an error as a failure to authenticate, which
// happens before a request is sent, rather than as the failure of a request.
var errOIDCTokenExchange = errors.New("exchanging OIDC token for a Vercel access token")

// OIDCConfig configures a Client to authenticate by exchanging an OIDC token
// from an external identity provider for a short-lived Vercel access token.
type OIDCConfig struct {
	// ClientID is the Vercel app the access token is issued for. The team needs
	// an OIDC federation policy for this app that matches the OIDC token.
	ClientID string
	// Team is the ID or slug of the team the access token is scoped to.
	Team string
	// PolicyID selects the OIDC federation policy to use. It is only needed
	// when more than one of the team's policies matches the OIDC token.
	PolicyID string
	// SubjectToken returns the OIDC token to exchange. It is called on every
	// exchange, so it can return a newer token once the previous one expired.
	SubjectToken func() (string, error)
}

type oidcTokenSource struct {
	config OIDCConfig

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
	refreshAt   time.Time
}

// NewWithOIDC creates a new instance of Client that authenticates with access
// tokens obtained by OIDC token exchange. The OIDC token is exchanged before
// the first request, and again whenever the access token is about to expire.
func NewWithOIDC(config OIDCConfig) *Client {
	c := New("")
	c.oidc = &oidcTokenSource{config: config}
	return c
}

// Authenticate makes sure the client holds a valid access token, exchanging
// the OIDC token when needed. It does nothing for a client using an API token.
func (c *Client) Authenticate(ctx context.Context) error {
	_, err := c.accessToken(ctx)
	return err
}

// accessToken returns the token used to authorize requests to the Vercel API.
func (c *Client) accessToken(ctx context.Context) (string, error) {
	if c.oidc == nil {
		return c.token, nil
	}

	// Holding the lock while exchanging makes concurrent requests wait for a
	// single exchange instead of each starting their own.
	c.oidc.mu.Lock()
	defer c.oidc.mu.Unlock()
	if c.oidc.accessToken != "" && time.Now().Before(c.oidc.refreshAt) {
		return c.oidc.accessToken, nil
	}

	accessToken, expiresIn, err := c.exchangeOIDCToken(ctx)
	now := time.Now()
	if err != nil {
		err = fmt.Errorf("%w: %w", errOIDCTokenExchange, err)
		if c.oidc.accessToken == "" || !now.Before(c.oidc.expiresAt) {
			return "", err
		}
		// The exchange happens ahead of the expiry, so the current access token
		// is still valid: keep using it, and try again shortly.
		tflog.Warn(ctx, "could not renew the Vercel access token, using the current one until it expires", map[string]any{
			"error": err.Error(),
		})
		c.oidc.refreshAt = now.Add(min(oidcTokenRefreshRetryInterval, c.oidc.expiresAt.Sub(now)))
		return c.oidc.accessToken, nil
	}
	c.oidc.accessToken = accessToken
	c.oidc.expiresAt = now.Add(expiresIn)
	c.oidc.refreshAt = c.oidc.expiresAt.Add(-min(oidcTokenRefreshLeeway, expiresIn/2))
	return accessToken, nil
}

func (c *Client) exchangeOIDCToken(ctx context.Context) (accessToken string, expiresIn time.Duration, err error) {
	config := c.oidc.config
	subjectToken, err := config.SubjectToken()
	if err != nil {
		return "", 0, err
	}
	if subjectToken == "" {
		return "", 0, errors.New("the OIDC token is empty")
	}

	form := url.Values{
		"grant_type":           {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"client_id":            {config.ClientID},
		"subject_token":        {subjectToken},
		"subject_token_type":   {"urn:ietf:params:oauth:token-type:id_token"},
		"requested_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		"team_id_or_slug":      {config.Team},
	}
	if config.PolicyID != "" {
		form.Set("policy_id", config.PolicyID)
	}

	endpoint := c.baseURL + oidcTokenExchangePath
	tflog.Info(ctx, "exchanging OIDC token for a Vercel access token", map[string]any{
		"url":       endpoint,
		"team":      config.Team,
		"policy_id": config.PolicyID,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", fmt.Sprintf("terraform-provider-vercel/%s", version))

	resp, err := c.http().Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("error doing http request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("error reading response body: %w", err)
	}

	// Errors are deliberately not an APIError, and response bodies are left out
	// of them: resources must not mistake a failed exchange for the status of
	// their own request, and a response can contain the access token.
	if resp.StatusCode != http.StatusOK {
		var oauthErr struct {
			Code        string `json:"error"`
			Description string `json:"error_description"`
		}
		if err := json.Unmarshal(body, &oauthErr); err == nil && oauthErr.Code != "" {
			return "", 0, fmt.Errorf(
				"%s: %s - %s (check that the OIDC token has not expired, and that its issuer and claims match a single OIDC federation policy of the team, or the one with the policy ID given)",
				resp.Status, oauthErr.Code, oauthErr.Description,
			)
		}
		return "", 0, errors.New(resp.Status)
	}

	var token struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &token); err != nil || token.AccessToken == "" {
		return "", 0, errors.New("the response does not contain an access token")
	}
	return token.AccessToken, time.Duration(token.ExpiresIn) * time.Second, nil
}
