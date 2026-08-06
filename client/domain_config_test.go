package client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGetDomainConfigIncludesMisconfigured(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got, want := r.URL.Path, "/v6/domains/www.example.com/config"; got != want {
			t.Fatalf("request path = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("projectIdOrName"), "prj_123"; got != want {
			t.Fatalf("projectIdOrName = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("teamId"), "team_123"; got != want {
			t.Fatalf("teamId = %q, want %q", got, want)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{
			"misconfigured": true,
			"recommendedCNAME": [{"rank": 1, "value": "example.vercel-dns-017.com"}],
			"recommendedIPv4": [{"rank": 1, "value": ["76.76.21.21"]}]
		}`)),
		}, nil
	})}

	testClient := New("INVALID").WithBaseURL("https://api.vercel.test")
	testClient.client = httpClient
	got, err := testClient.GetDomainConfig(
		context.Background(),
		"www.example.com",
		"prj_123",
		"team_123",
	)
	if err != nil {
		t.Fatalf("GetDomainConfig() error = %v", err)
	}
	if !got.Misconfigured {
		t.Fatal("GetDomainConfig().Misconfigured = false, want true")
	}
	if got.RecommendedCNAME != "example.vercel-dns-017.com" {
		t.Fatalf("GetDomainConfig().RecommendedCNAME = %q, want %q", got.RecommendedCNAME, "example.vercel-dns-017.com")
	}
	if len(got.RecommendedIPv4s) != 1 || got.RecommendedIPv4s[0] != "76.76.21.21" {
		t.Fatalf("GetDomainConfig().RecommendedIPv4s = %v, want [76.76.21.21]", got.RecommendedIPv4s)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
