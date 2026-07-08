package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
		})
	}
}
