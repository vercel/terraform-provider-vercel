package vercel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestSharedEnvironmentVariableProjectLinkReadRemovesDeletedVariable(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name        string
		status      int
		body        string
		wantRemoved bool
		wantError   bool
	}{
		{
			name:   "linked",
			status: http.StatusOK,
			body:   `{"id":"env_123","ownerId":"team_123","projectId":["prj_123"]}`,
		},
		{
			name:        "unlinked",
			status:      http.StatusOK,
			body:        `{"id":"env_123","ownerId":"team_123","projectId":["prj_other"]}`,
			wantRemoved: true,
		},
		{
			name:        "variable deleted",
			status:      http.StatusNotFound,
			body:        `{"error":{"code":"not_found","message":"Environment variable not found"}}`,
			wantRemoved: true,
		},
		{
			name:      "server error",
			status:    http.StatusBadRequest,
			body:      `{"error":{"code":"bad_request","message":"boom"}}`,
			wantError: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/v1/env/env_123" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			res := &sharedEnvironmentVariableProjectLinkResource{
				client: client.New("abcdefghijklmnopqrstuvwx").WithBaseURL(server.URL),
			}

			schemaResp := &resource.SchemaResponse{}
			res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

			state := tfsdk.State{Schema: schemaResp.Schema}
			diags := state.Set(ctx, SharedEnvironmentVariableProjectLink{
				ID:                          types.StringValue("team_123/env_123"),
				SharedEnvironmentVariableID: types.StringValue("env_123"),
				ProjectID:                   types.StringValue("prj_123"),
				TeamID:                      types.StringValue("team_123"),
			})
			if diags.HasError() {
				t.Fatalf("set state: %s", diags.Errors())
			}

			readResp := resource.ReadResponse{State: state}
			res.Read(ctx, resource.ReadRequest{State: state}, &readResp)

			if got := readResp.Diagnostics.HasError(); got != tt.wantError {
				t.Fatalf("error = %t, want %t: %s", got, tt.wantError, readResp.Diagnostics.Errors())
			}
			if got := readResp.State.Raw.IsNull(); got != tt.wantRemoved {
				t.Fatalf("removed from state = %t, want %t", got, tt.wantRemoved)
			}
		})
	}
}
