package vercel

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestWebhookImportState(t *testing.T) {
	t.Parallel()

	const (
		webhookID = "hook_123"
		teamID    = "team_123"
	)

	for _, tt := range []struct {
		name           string
		importID       string
		configuredTeam client.Team
	}{
		{
			name:     "team and webhook ID",
			importID: teamID + "/" + webhookID,
		},
		{
			name:           "webhook ID with provider team",
			importID:       webhookID,
			configuredTeam: client.Team{ID: teamID},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				if r.URL.Path != "/v1/webhooks/"+webhookID {
					t.Fatalf("path = %s, want /v1/webhooks/%s", r.URL.Path, webhookID)
				}
				if got := r.URL.Query().Get("teamId"); got != teamID {
					t.Fatalf("teamId = %q, want %q", got, teamID)
				}

				_, _ = fmt.Fprint(w, `{
					"id": "hook_123",
					"ownerId": "team_123",
					"url": "https://example.com/webhook",
					"events": ["deployment.succeeded", "deployment.promoted"],
					"projectIds": ["prj_123"]
				}`)
			}))
			defer server.Close()

			imported := importWebhookState(t, server.URL, tt.configuredTeam, tt.importID)
			if imported.ID.ValueString() != webhookID {
				t.Fatalf("id = %q, want %q", imported.ID.ValueString(), webhookID)
			}
			if imported.TeamID.ValueString() != teamID {
				t.Fatalf("team_id = %q, want %q", imported.TeamID.ValueString(), teamID)
			}
			if imported.Endpoint.ValueString() != "https://example.com/webhook" {
				t.Fatalf("endpoint = %q, want https://example.com/webhook", imported.Endpoint.ValueString())
			}
			if !imported.Secret.IsNull() {
				t.Fatalf("secret = %s, want null", imported.Secret)
			}

			var projectIDs []string
			diags := imported.ProjectIDs.ElementsAs(context.Background(), &projectIDs, false)
			if diags.HasError() || len(projectIDs) != 1 || projectIDs[0] != "prj_123" {
				t.Fatalf("project_ids = %v, diagnostics = %v", projectIDs, diags)
			}
			var events []string
			diags = imported.Events.ElementsAs(context.Background(), &events, false)
			slices.Sort(events)
			if diags.HasError() || !slices.Equal(events, []string{"deployment.promoted", "deployment.succeeded"}) {
				t.Fatalf("events = %v, diagnostics = %v", events, diags)
			}
		})
	}
}

func TestWebhookImportStateAPIErrors(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name             string
		status           int
		expectDiagnostic bool
	}{
		{name: "not found removes resource", status: http.StatusNotFound},
		{name: "API error returns diagnostic", status: http.StatusInternalServerError, expectDiagnostic: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = fmt.Fprint(w, `{"error":{"code":"test_error","message":"test error"}}`)
			}))
			defer server.Close()

			res, resp := webhookImportResponse(t, server.URL, client.Team{})
			res.ImportState(context.Background(), resource.ImportStateRequest{ID: "team_123/hook_123"}, resp)

			if resp.Diagnostics.HasError() != tt.expectDiagnostic {
				t.Fatalf("diagnostics error = %v, want %v: %v", resp.Diagnostics.HasError(), tt.expectDiagnostic, resp.Diagnostics)
			}
			if !tt.expectDiagnostic && !resp.State.Raw.IsNull() {
				t.Fatalf("state = %s, want null", resp.State.Raw)
			}
		})
	}
}

func importWebhookState(t *testing.T, baseURL string, team client.Team, importID string) Webhook {
	t.Helper()

	res, resp := webhookImportResponse(t, baseURL, team)
	res.ImportState(context.Background(), resource.ImportStateRequest{ID: importID}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %v", resp.Diagnostics)
	}

	var state Webhook
	diags := resp.State.Get(context.Background(), &state)
	if diags.HasError() {
		t.Fatalf("get state diagnostics: %v", diags)
	}
	return state
}

func webhookImportResponse(t *testing.T, baseURL string, team client.Team) (*webhookResource, *resource.ImportStateResponse) {
	t.Helper()

	res := &webhookResource{
		client: client.New("token").WithBaseURL(baseURL).WithTeam(team),
	}
	schemaResp := &resource.SchemaResponse{}
	res.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", schemaResp.Diagnostics)
	}

	return res, &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
}
