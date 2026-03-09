package vercel

import (
	"context"
	"testing"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

func TestFindLiveBulkRedirectVersion(t *testing.T) {
	t.Parallel()

	live := client.BulkRedirectVersion{ID: "ver_live", IsLive: true}
	version, ok := findLiveBulkRedirectVersion([]client.BulkRedirectVersion{
		{ID: "ver_staging", IsStaging: true},
		live,
	})
	if !ok {
		t.Fatal("expected to find a live version")
	}
	if version.ID != live.ID {
		t.Fatalf("expected live version %q, got %q", live.ID, version.ID)
	}
}

func TestFlattenAndExpandBulkRedirects(t *testing.T) {
	t.Parallel()

	statusCode := int64(308)
	caseSensitive := false
	query := true

	list := flattenBulkRedirects([]client.BulkRedirect{
		{
			Source:        "/old-path",
			Destination:   "/new-path",
			StatusCode:    &statusCode,
			CaseSensitive: &caseSensitive,
			Query:         &query,
		},
	})

	redirects, diags := expandBulkRedirects(context.Background(), list)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(redirects) != 1 {
		t.Fatalf("expected 1 redirect, got %d", len(redirects))
	}

	redirect := redirects[0]
	if redirect.Source != "/old-path" {
		t.Fatalf("expected source /old-path, got %q", redirect.Source)
	}
	if redirect.Destination != "/new-path" {
		t.Fatalf("expected destination /new-path, got %q", redirect.Destination)
	}
	if redirect.StatusCode == nil || *redirect.StatusCode != statusCode {
		t.Fatalf("expected status code %d, got %#v", statusCode, redirect.StatusCode)
	}
	if redirect.CaseSensitive == nil || *redirect.CaseSensitive != caseSensitive {
		t.Fatalf("expected case_sensitive %t, got %#v", caseSensitive, redirect.CaseSensitive)
	}
	if redirect.Query == nil || *redirect.Query != query {
		t.Fatalf("expected query %t, got %#v", query, redirect.Query)
	}
}
