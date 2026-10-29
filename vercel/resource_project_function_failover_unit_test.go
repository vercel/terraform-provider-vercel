package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestProjectFunctionFailoverImportReadUpdate(t *testing.T) {
	boolPointer := func(v bool) *bool { return &v }
	for _, tc := range []struct {
		name          string
		getFailover   string
		patchFailover string
		configured    *bool
		wantImported  types.Bool
		wantRequest   string
		wantFinal     types.Bool
	}{
		{
			name:          "live value true with unconfigured attribute",
			getFailover:   "true",
			patchFailover: "true",
			wantImported:  types.BoolValue(true),
			wantRequest:   "true",
			wantFinal:     types.BoolValue(true),
		},
		{
			name:         "inherited value with unconfigured attribute",
			wantImported: types.BoolNull(),
			wantRequest:  "",
			wantFinal:    types.BoolNull(),
		},
		{
			name:          "explicit false",
			patchFailover: "false",
			configured:    boolPointer(false),
			wantImported:  types.BoolNull(),
			wantRequest:   "false",
			wantFinal:     types.BoolValue(false),
		},
		{
			name:         "explicit false survives an omitted response field",
			getFailover:  "false",
			configured:   boolPointer(false),
			wantImported: types.BoolValue(false),
			wantRequest:  "false",
			wantFinal:    types.BoolValue(false),
		},
		{
			name:         "explicit true survives an omitted response field",
			configured:   boolPointer(true),
			wantImported: types.BoolNull(),
			wantRequest:  "true",
			wantFinal:    types.BoolValue(true),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			updates := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				failover := tc.getFailover
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/v10/projects/prj_123":
				case req.Method == http.MethodPatch && req.URL.Path == "/v9/projects/prj_123":
					updates++
					var body map[string]json.RawMessage
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Error(err)
						http.Error(w, "invalid JSON", http.StatusBadRequest)
						return
					}
					if got := string(body["serverlessFunctionZeroConfigFailover"]); got != tc.wantRequest {
						t.Errorf("serverlessFunctionZeroConfigFailover = %q, want %q", got, tc.wantRequest)
					}
					failover = tc.patchFailover
				default:
					t.Errorf("unexpected request: %s %s", req.Method, req.URL)
					http.NotFound(w, req)
					return
				}
				if failover != "" {
					failover = fmt.Sprintf(`,"serverlessFunctionZeroConfigFailover":%s`, failover)
				}
				fmt.Fprintf(w, `{"id":"prj_123","name":"example","accountId":"team_123","nodeVersion":"22.x"%s}`, failover)
			}))
			defer server.Close()

			ctx := context.Background()
			r := &projectResource{client: client.New("test").WithBaseURL(server.URL)}
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			importResp := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			r.ImportState(ctx, resource.ImportStateRequest{ID: "prj_123"}, &importResp)
			if importResp.Diagnostics.HasError() {
				t.Fatal(importResp.Diagnostics)
			}
			failoverPath := path.Root("function_failover")
			var imported types.Bool
			if diags := importResp.State.GetAttribute(ctx, failoverPath, &imported); diags.HasError() {
				t.Fatal(diags)
			}
			if !imported.Equal(tc.wantImported) {
				t.Fatalf("imported function_failover = %s, want %s", imported, tc.wantImported)
			}

			// Mirror UseNonNullStateForUnknown: an unconfigured attribute is planned as
			// the prior state value, or as unknown when that value is null.
			planned := imported
			if tc.configured != nil {
				planned = types.BoolValue(*tc.configured)
			} else if imported.IsNull() {
				planned = types.BoolUnknown()
			}
			plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: importResp.State.Raw}
			if diags := plan.SetAttribute(ctx, failoverPath, planned); diags.HasError() {
				t.Fatal(diags)
			}
			updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			r.Update(ctx, resource.UpdateRequest{State: importResp.State, Plan: plan}, &updateResp)
			if updateResp.Diagnostics.HasError() {
				t.Fatal(updateResp.Diagnostics)
			}
			if updates != 1 {
				t.Fatalf("updates = %d, want 1", updates)
			}
			var final types.Bool
			if diags := updateResp.State.GetAttribute(ctx, failoverPath, &final); diags.HasError() {
				t.Fatal(diags)
			}
			if !final.Equal(tc.wantFinal) {
				t.Fatalf("function_failover after update = %s, want %s", final, tc.wantFinal)
			}
		})
	}
}
