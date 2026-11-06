package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestProjectProtectionBypassCreatePreservesPlannedNoteWhenAPIOmitsIt(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name           string
		note           types.String
		expectBodyNote bool
	}{
		{
			name:           "non-empty note",
			note:           types.StringValue("note"),
			expectBodyNote: true,
		},
		{
			name:           "empty note",
			note:           types.StringValue(""),
			expectBodyNote: true,
		},
		{
			name: "omitted note",
			note: types.StringNull(),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			const (
				projectID = "prj_123"
				secret    = "abcdefghijklmnopqrstuvwxyz123456"
			)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPatch {
					t.Fatalf("method = %s, want PATCH", r.Method)
				}
				if r.URL.Path != fmt.Sprintf("/v10/projects/%s/protection-bypass", projectID) {
					t.Fatalf("path = %s, want /v10/projects/%s/protection-bypass", r.URL.Path, projectID)
				}

				var body struct {
					Generate struct {
						Secret string  `json:"secret"`
						Note   *string `json:"note"`
					} `json:"generate"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode request body: %s", err)
				}

				if body.Generate.Secret != secret {
					t.Fatalf("generate.secret = %q, want %q", body.Generate.Secret, secret)
				}
				if tt.expectBodyNote {
					if body.Generate.Note == nil {
						t.Fatal("generate.note was omitted")
					}
					if *body.Generate.Note != tt.note.ValueString() {
						t.Fatalf("generate.note = %q, want %q", *body.Generate.Note, tt.note.ValueString())
					}
				} else if body.Generate.Note != nil {
					t.Fatalf("generate.note = %q, want omitted", *body.Generate.Note)
				}

				_, _ = fmt.Fprintf(w, `{
					"protectionBypass": {
						%q: {
							"scope": "automation-bypass",
							"isEnvVar": true,
							"createdAt": 123,
							"createdBy": "user_123"
						}
					}
				}`, secret)
			}))
			defer server.Close()

			res := &projectProtectionBypassResource{
				client: client.New("abcdefghijklmnopqrstuvwx").WithBaseURL(server.URL),
			}

			schemaResp := &resource.SchemaResponse{}
			res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

			plan := tfsdk.Plan{Schema: schemaResp.Schema}
			diags := plan.Set(ctx, ProjectProtectionBypass{
				ID:        types.StringUnknown(),
				ProjectID: types.StringValue(projectID),
				TeamID:    types.StringNull(),
				Secret:    types.StringValue(secret),
				Note:      tt.note,
				IsEnvVar:  types.BoolNull(),
				Scope:     types.StringUnknown(),
				CreatedAt: types.Int64Unknown(),
				CreatedBy: types.StringUnknown(),
			})
			if diags.HasError() {
				t.Fatalf("set plan: %s", diags.Errors())
			}

			createResp := resource.CreateResponse{
				State: tfsdk.State{Schema: schemaResp.Schema},
			}
			res.Create(ctx, resource.CreateRequest{Plan: plan}, &createResp)
			if createResp.Diagnostics.HasError() {
				t.Fatalf("create diagnostics: %s", createResp.Diagnostics.Errors())
			}

			var state ProjectProtectionBypass
			diags = createResp.State.Get(ctx, &state)
			if diags.HasError() {
				t.Fatalf("get state: %s", diags.Errors())
			}
			if !state.Note.Equal(tt.note) {
				t.Fatalf("state note = %s, want %s", state.Note, tt.note)
			}
			if strings.Contains(state.ID.ValueString(), secret) {
				t.Fatal("resource ID exposes the bypass secret")
			}
		})
	}
}

func TestProjectProtectionBypassUpgradeRemovesSecretFromID(t *testing.T) {
	ctx := context.Background()
	res := &projectProtectionBypassResource{}
	upgrade := res.UpgradeState(ctx)[0]
	prior := convertBypass("prj_123", "team_123", "abcdefghijklmnopqrstuvwxyz123456", client.ProtectionBypass{
		Scope: "automation-bypass", CreatedAt: 123, CreatedBy: "user_123",
	})
	prior.ID = types.StringValue(prior.ProjectID.ValueString() + "/" + prior.Secret.ValueString())
	prior.Note = types.StringValue("keep this note")
	prior.IsEnvVar = types.BoolValue(true)
	priorState := tfsdk.State{Schema: *upgrade.PriorSchema}
	if diags := priorState.Set(ctx, prior); diags.HasError() {
		t.Fatal(diags)
	}
	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Schema.Version != 1 {
		t.Fatalf("schema version = %d, want 1", schemaResp.Schema.Version)
	}
	resp := resource.UpgradeStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	upgrade.StateUpgrader(ctx, resource.UpgradeStateRequest{State: &priorState}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var upgraded ProjectProtectionBypass
	if diags := resp.State.Get(ctx, &upgraded); diags.HasError() {
		t.Fatal(diags)
	}
	if upgraded.ID.Equal(prior.ID) || strings.Contains(upgraded.ID.ValueString(), prior.Secret.ValueString()) {
		t.Fatal("upgraded ID still exposes the secret")
	}
	// Compare the full state after restoring the ID to catch lost metadata.
	upgraded.ID = prior.ID
	if diags := resp.State.Set(ctx, upgraded); diags.HasError() {
		t.Fatal(diags)
	}
	if !resp.State.Raw.Equal(priorState.Raw) {
		t.Fatal("upgrade changed attributes other than ID")
	}
}

func TestProjectProtectionBypassOpaqueIDLifecycle(t *testing.T) {
	ctx := context.Background()
	const projectID = "prj_123"
	const secret = "abcdefghijklmnopqrstuvwxyz123456"
	patches := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v10/projects/"+projectID:
		case req.Method == http.MethodPatch && req.URL.Path == "/v10/projects/"+projectID+"/protection-bypass":
			var body struct {
				Update struct{ Secret string } `json:"update"`
				Revoke struct{ Secret string } `json:"revoke"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body.Update.Secret != secret && body.Revoke.Secret != secret {
				t.Error("API mutation did not use the sensitive secret")
			}
			patches++
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		fmt.Fprintf(w, `{"protectionBypass":{%q:{"scope":"automation-bypass","isEnvVar":false,"note":"updated","createdAt":123,"createdBy":"user_123"}}}`, secret)
	}))
	defer server.Close()
	res := &projectProtectionBypassResource{client: client.New("abcdefghijklmnopqrstuvwx").WithBaseURL(server.URL)}
	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	var imported tfsdk.State
	for _, importID := range []string{projectID + "/" + secret, "team_123/" + projectID + "/" + secret} {
		resp := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		res.ImportState(ctx, resource.ImportStateRequest{ID: importID}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		imported = resp.State
	}
	var model ProjectProtectionBypass
	if diags := imported.Get(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	if strings.Contains(model.ID.ValueString(), secret) || model.Secret.ValueString() != secret {
		t.Fatal("import did not separate the visible ID from the sensitive secret")
	}
	id := model.ID
	if id.Equal(projectProtectionBypassID(projectID, "123456abcdefghijklmnopqrstuvwxyz")) || id.Equal(projectProtectionBypassID("prj_other", secret)) {
		t.Fatal("different bypasses have the same ID")
	}
	readResp := resource.ReadResponse{State: imported}
	res.Read(ctx, resource.ReadRequest{State: imported}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	if !readResp.State.Raw.Equal(imported.Raw) {
		t.Fatal("refresh changed imported state")
	}
	model.Note = types.StringValue("updated note")
	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := plan.Set(ctx, model); diags.HasError() {
		t.Fatal(diags)
	}
	updateResp := resource.UpdateResponse{State: imported}
	res.Update(ctx, resource.UpdateRequest{State: imported, Plan: plan}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	if diags := updateResp.State.Get(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	if !model.ID.Equal(id) {
		t.Fatal("updating metadata changed the resource ID")
	}
	var deleteResp resource.DeleteResponse
	res.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatal(deleteResp.Diagnostics)
	}
	if patches != 2 {
		t.Fatalf("API mutations = %d, want one update and one revoke", patches)
	}
}

func TestProjectProtectionBypassInvalidImportDoesNotExposeSecret(t *testing.T) {
	const secret = "abcdefghijklmnopqrstuvwxyz123456"
	var resp resource.ImportStateResponse
	res := &projectProtectionBypassResource{}
	res.ImportState(context.Background(), resource.ImportStateRequest{ID: "team/project/" + secret + "/extra"}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an invalid import ID error")
	}
	for _, diag := range resp.Diagnostics {
		if strings.Contains(diag.Detail(), secret) {
			t.Fatal("import diagnostic exposes the secret")
		}
	}
}
