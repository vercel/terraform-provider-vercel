package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// newOIDCTestServer exchanges the nth OIDC token for "access-n", and reports
// the form of every exchange and the Authorization header of every API request.
func newOIDCTestServer(t *testing.T) (server *httptest.Server, exchanges *[]url.Values, authorizations *[]string) {
	t.Helper()
	exchanges, authorizations = &[]url.Values{}, &[]string{}
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != oidcTokenExchangePath {
			*authorizations = append(*authorizations, r.Header.Get("Authorization"))
			fmt.Fprint(w, `{"id":"team_123"}`)
			return
		}
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("exchange = %s with content type %q, want a form POST", r.Method, r.Header.Get("Content-Type"))
		}
		if authorization := r.Header.Get("Authorization"); authorization != "" {
			t.Errorf("exchange Authorization = %q, want none", authorization)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm() error = %v", err)
		}
		*exchanges = append(*exchanges, r.PostForm)
		fmt.Fprintf(w, `{"access_token":"access-%d","token_type":"Bearer","expires_in":600}`, len(*exchanges))
	}))
	t.Cleanup(server.Close)
	return server, exchanges, authorizations
}

func TestOIDCClientExchangesTokenOnceWhileAccessTokenIsValid(t *testing.T) {
	server, exchanges, authorizations := newOIDCTestServer(t)
	c := NewWithOIDC(OIDCConfig{
		ClientID:     "cl_123",
		Team:         "acme",
		PolicyID:     "pol_123",
		SubjectToken: func() (string, error) { return "oidc-token", nil },
	}).WithBaseURL(server.URL)

	for range 2 {
		if _, err := c.GetTeam(context.Background(), "team_123"); err != nil {
			t.Fatalf("GetTeam() error = %v", err)
		}
	}

	if len(*exchanges) != 1 {
		t.Fatalf("exchanges = %d, want 1", len(*exchanges))
	}
	want := url.Values{
		"grant_type":           {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"client_id":            {"cl_123"},
		"subject_token":        {"oidc-token"},
		"subject_token_type":   {"urn:ietf:params:oauth:token-type:id_token"},
		"requested_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		"team_id_or_slug":      {"acme"},
		"policy_id":            {"pol_123"},
	}
	if got := (*exchanges)[0]; got.Encode() != want.Encode() {
		t.Fatalf("exchange form = %v, want %v", got, want)
	}
	if got := strings.Join(*authorizations, ","); got != "Bearer access-1,Bearer access-1" {
		t.Fatalf("authorizations = %q, want both requests to use access-1", got)
	}
}

func TestOIDCClientExchangesNewTokenWhenAccessTokenIsAboutToExpire(t *testing.T) {
	server, exchanges, authorizations := newOIDCTestServer(t)
	subjectTokens := 0
	c := NewWithOIDC(OIDCConfig{
		ClientID: "cl_123",
		Team:     "acme",
		SubjectToken: func() (string, error) {
			subjectTokens++
			return fmt.Sprintf("oidc-token-%d", subjectTokens), nil
		},
	}).WithBaseURL(server.URL)

	if err := c.Authenticate(context.Background()); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if untilRefresh := time.Until(c.oidc.refreshAt); untilRefresh < 8*time.Minute || untilRefresh > 9*time.Minute {
		t.Fatalf("refresh in %s, want it a minute before the access token expires in 10m", untilRefresh)
	}
	c.oidc.refreshAt = time.Now()
	if _, err := c.GetTeam(context.Background(), "team_123"); err != nil {
		t.Fatalf("GetTeam() error = %v", err)
	}

	if len(*exchanges) != 2 {
		t.Fatalf("exchanges = %d, want 2", len(*exchanges))
	}
	if got := (*exchanges)[1].Get("subject_token"); got != "oidc-token-2" {
		t.Fatalf("second exchange subject_token = %q, want a newly read oidc-token-2", got)
	}
	if (*exchanges)[1].Has("policy_id") {
		t.Fatalf("exchange form = %v, want no policy_id when none is configured", (*exchanges)[1])
	}
	if got := strings.Join(*authorizations, ","); got != "Bearer access-2" {
		t.Fatalf("authorizations = %q, want the request to use access-2", got)
	}
}

func TestOIDCClientReportsFailedExchange(t *testing.T) {
	tests := []struct {
		name         string
		subjectToken func() (string, error)
		status       int
		body         string
		want         string
	}{
		{
			name:   "OAuth error",
			status: http.StatusBadRequest,
			body:   `{"error":"invalid_request","error_description":"Not authorized."}`,
			want:   "400 Bad Request: invalid_request - Not authorized. (check that the OIDC token has not expired, and that its issuer and claims match a single OIDC federation policy of the team, or the one with the policy ID given)",
		},
		{
			name:   "status not mistaken for the resource being missing",
			status: http.StatusNotFound,
			body:   `{"error":{"code":"not_found","message":"Not found"}}`,
			want:   "404 Not Found",
		},
		{
			name:   "response without an access token",
			status: http.StatusOK,
			body:   `{"token_type":"Bearer"}`,
			want:   "the response does not contain an access token",
		},
		{
			name:         "unreadable OIDC token",
			subjectToken: func() (string, error) { return "", fmt.Errorf("reading the OIDC token file: missing") },
			want:         "reading the OIDC token file: missing",
		},
		{
			name:         "empty OIDC token",
			subjectToken: func() (string, error) { return "", nil },
			want:         "the OIDC token is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != oidcTokenExchangePath {
					t.Errorf("request to %s, want none after a failed exchange", r.URL.Path)
				}
				w.WriteHeader(tt.status)
				fmt.Fprint(w, tt.body)
			}))
			t.Cleanup(server.Close)
			subjectToken := tt.subjectToken
			if subjectToken == nil {
				subjectToken = func() (string, error) { return "oidc-token", nil }
			}
			c := NewWithOIDC(OIDCConfig{ClientID: "cl_123", Team: "acme", SubjectToken: subjectToken}).WithBaseURL(server.URL)

			want := "exchanging OIDC token for a Vercel access token: " + tt.want
			_, err := c.GetTeam(context.Background(), "team_123")
			if err == nil || err.Error() != want {
				t.Fatalf("GetTeam() error = %v, want %q", err, want)
			}
			if NotFound(err) {
				t.Fatalf("NotFound(%v) = true, want false", err)
			}

			err = c.doRequest(clientRequest{ctx: context.Background(), method: http.MethodGet, url: server.URL, sensitiveResponse: true}, nil)
			if err == nil || err.Error() != want {
				t.Fatalf("doRequest() error for a sensitive response = %v, want %q", err, want)
			}
		})
	}
}

func TestOIDCClientKeepsAccessTokenUntilItExpiresWhenRenewalFails(t *testing.T) {
	exchanges := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != oidcTokenExchangePath {
			if authorization := r.Header.Get("Authorization"); authorization != "Bearer access-1" {
				t.Errorf("Authorization = %q, want the access token still valid", authorization)
			}
			fmt.Fprint(w, `{"id":"team_123"}`)
			return
		}
		exchanges++
		if exchanges > 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, `{"access_token":"access-1","token_type":"Bearer","expires_in":600}`)
	}))
	t.Cleanup(server.Close)
	c := NewWithOIDC(OIDCConfig{
		ClientID:     "cl_123",
		Team:         "acme",
		SubjectToken: func() (string, error) { return "oidc-token", nil },
	}).WithBaseURL(server.URL)
	if err := c.Authenticate(context.Background()); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	c.oidc.refreshAt = time.Now()
	for range 2 {
		if _, err := c.GetTeam(context.Background(), "team_123"); err != nil {
			t.Fatalf("GetTeam() error = %v, want none while the access token is valid", err)
		}
	}
	if exchanges != 2 {
		t.Fatalf("exchanges = %d, want 2: the failed renewal is not retried right away", exchanges)
	}

	c.oidc.expiresAt, c.oidc.refreshAt = time.Now(), time.Now()
	_, err := c.GetTeam(context.Background(), "team_123")
	if want := "exchanging OIDC token for a Vercel access token: 503 Service Unavailable"; err == nil || err.Error() != want {
		t.Fatalf("GetTeam() error = %v, want %q once the access token expired", err, want)
	}
}
