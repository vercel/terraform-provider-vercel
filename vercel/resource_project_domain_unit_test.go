package vercel

import (
	"context"
	"testing"

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
	})

	if got := result.Verified.ValueBool(); got {
		t.Fatalf("Verified = %t, want false", got)
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
