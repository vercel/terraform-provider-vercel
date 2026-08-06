package vercel

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestConvertResponseToProjectDomainIncludesVerification(t *testing.T) {
	redirectStatusCode := int64(308)
	redirect := "example.com"
	gitBranch := "main"
	customEnvironmentID := "env_123"

	result := convertResponseToProjectDomain(client.ProjectDomainResponse{
		Name:                "www.example.com",
		ProjectID:           "prj_123",
		TeamID:              "team_123",
		Redirect:            &redirect,
		RedirectStatusCode:  &redirectStatusCode,
		GitBranch:           &gitBranch,
		CustomEnvironmentID: &customEnvironmentID,
		Verified:            false,
		Verification: []client.ProjectDomainVerification{
			{
				Type:   "TXT",
				Domain: "_vercel.www.example.com",
				Value:  "vc-domain-verify=www.example.com,abc123",
				Reason: "pending_domain_verification",
			},
		},
	}, client.DomainConfigResponse{Misconfigured: true})

	if got := result.Verified.ValueBool(); got {
		t.Fatalf("Verified = %t, want false", got)
	}
	if got := result.Misconfigured.ValueBool(); !got {
		t.Fatalf("Misconfigured = %t, want true", got)
	}

	var verification []ProjectDomainVerification
	diags := result.Verification.ElementsAs(context.Background(), &verification, false)
	if diags.HasError() {
		t.Fatalf("Verification.ElementsAs() returned diagnostics: %v", diags)
	}
	if len(verification) != 1 {
		t.Fatalf("len(Verification) = %d, want 1", len(verification))
	}

	first := verification[0]
	if got := first.Type.ValueString(); got != "TXT" {
		t.Fatalf("Verification[0].Type = %q, want TXT", got)
	}
	if got := first.Domain.ValueString(); got != "_vercel.www.example.com" {
		t.Fatalf("Verification[0].Domain = %q, want _vercel.www.example.com", got)
	}
	if got := first.Value.ValueString(); got != "vc-domain-verify=www.example.com,abc123" {
		t.Fatalf("Verification[0].Value = %q, want vc-domain-verify=www.example.com,abc123", got)
	}
	if got := first.Reason.ValueString(); got != "pending_domain_verification" {
		t.Fatalf("Verification[0].Reason = %q, want pending_domain_verification", got)
	}
}

func TestProjectDomainReady(t *testing.T) {
	tests := []struct {
		name          string
		verified      bool
		misconfigured bool
		want          bool
	}{
		{name: "ready", verified: true, misconfigured: false, want: true},
		{name: "ownership pending", verified: false, misconfigured: false, want: false},
		{name: "DNS misconfigured", verified: true, misconfigured: true, want: false},
		{name: "both pending", verified: false, misconfigured: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := projectDomainReady(
				client.ProjectDomainResponse{Verified: tt.verified},
				client.DomainConfigResponse{Misconfigured: tt.misconfigured},
			)
			if got != tt.want {
				t.Fatalf("projectDomainReady() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestProjectDomainNotReadyReasonIncludesDNSRecommendations(t *testing.T) {
	got := projectDomainNotReadyReason(
		client.ProjectDomainResponse{Verified: true},
		client.DomainConfigResponse{
			Misconfigured:    true,
			RecommendedCNAME: "example.vercel-dns-017.com",
			RecommendedIPv4s: []string{"76.76.21.21"},
		},
	)
	want := `DNS configuration is still invalid; recommended values: CNAME "example.vercel-dns-017.com", IPv4 addresses 76.76.21.21`
	if got != want {
		t.Fatalf("projectDomainNotReadyReason() = %q, want %q", got, want)
	}
}

func TestWaitForProjectDomainReadyChecksDNSConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v9/projects/prj_123/domains/www.example.com":
			fmt.Fprintln(w, `{"name":"www.example.com","projectId":"prj_123","verified":true}`)
		case "/v6/domains/www.example.com/config":
			fmt.Fprintln(w, `{
				"misconfigured": true,
				"recommendedCNAME": [{"rank": 1, "value": "example.vercel-dns-017.com"}],
				"recommendedIPv4": []
			}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	testResource := &projectDomainResource{client: client.New("INVALID").WithBaseURL(server.URL)}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, _, err := testResource.waitForProjectDomainReady(ctx, "prj_123", "www.example.com", "team_123")
	if err == nil {
		t.Fatal("waitForProjectDomainReady() error = nil, want DNS misconfiguration error")
	}
	if !strings.Contains(err.Error(), "DNS configuration is still invalid") {
		t.Fatalf("waitForProjectDomainReady() error = %q, want DNS misconfiguration details", err)
	}
	if !strings.Contains(err.Error(), `CNAME "example.vercel-dns-017.com"`) {
		t.Fatalf("waitForProjectDomainReady() error = %q, want recommended CNAME", err)
	}
}
