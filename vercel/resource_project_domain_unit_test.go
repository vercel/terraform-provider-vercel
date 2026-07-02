package vercel

import (
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
	if len(result.Verification) != 1 {
		t.Fatalf("len(Verification) = %d, want 1", len(result.Verification))
	}

	verification := result.Verification[0]
	if got := verification.Type.ValueString(); got != "TXT" {
		t.Fatalf("Verification[0].Type = %q, want TXT", got)
	}
	if got := verification.Domain.ValueString(); got != "_vercel.www.example.com" {
		t.Fatalf("Verification[0].Domain = %q, want _vercel.www.example.com", got)
	}
	if got := verification.Value.ValueString(); got != "vc-domain-verify=www.example.com,abc123" {
		t.Fatalf("Verification[0].Value = %q, want vc-domain-verify=www.example.com,abc123", got)
	}
	if got := verification.Reason.ValueString(); got != "pending_domain_verification" {
		t.Fatalf("Verification[0].Reason = %q, want pending_domain_verification", got)
	}
}
