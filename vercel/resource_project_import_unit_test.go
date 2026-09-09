package vercel

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestProjectImportAndReadOwnership(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name           string
		importID       string
		configuredTeam client.Team
		accountID      string
		requestTeam    string
		wantTeam       string
	}{
		{
			name:      "bare ID discovers team owner",
			importID:  "prj_123",
			accountID: "team_123",
			wantTeam:  "team_123",
		},
		{
			name:        "explicit team",
			importID:    "team_123/prj_123",
			accountID:   "team_123",
			requestTeam: "team_123",
			wantTeam:    "team_123",
		},
		{
			name:           "provider team",
			importID:       "prj_123",
			configuredTeam: client.Team{ID: "team_123"},
			accountID:      "team_123",
			requestTeam:    "team_123",
			wantTeam:       "team_123",
		},
		{
			name:      "personal owner remains null",
			importID:  "prj_123",
			accountID: "user_123",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("method = %s, want GET", r.Method)
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				switch r.URL.Path {
				case "/v10/projects/prj_123":
					if got := r.URL.Query().Get("teamId"); got != tt.requestTeam {
						t.Errorf("project request teamId = %q, want %q", got, tt.requestTeam)
					}
					_, _ = fmt.Fprintf(w, `{"id":"prj_123","name":"test-project","accountId":%q}`, tt.accountID)
				case "/v8/projects/prj_123/env":
					if got := r.URL.Query().Get("teamId"); got != tt.wantTeam {
						t.Errorf("environment request teamId = %q, want discovered owner %q", got, tt.wantTeam)
					}
					_, _ = fmt.Fprint(w, `{"envs":[]}`)
				default:
					t.Errorf("unexpected path: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			res := &projectResource{client: client.New("token").WithBaseURL(server.URL).WithTeam(tt.configuredTeam)}
			schemaResp := &resource.SchemaResponse{}
			res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
			if schemaResp.Diagnostics.HasError() {
				t.Fatalf("schema diagnostics: %v", schemaResp.Diagnostics)
			}
			importResp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			res.ImportState(ctx, resource.ImportStateRequest{ID: tt.importID}, importResp)
			if importResp.Diagnostics.HasError() {
				t.Fatalf("import diagnostics: %v", importResp.Diagnostics)
			}
			assertProjectOwnership(t, importResp.State, tt.wantTeam)

			var prior Project
			if diags := importResp.State.Get(ctx, &prior); diags.HasError() {
				t.Fatalf("get state diagnostics: %v", diags)
			}
			// Older bare-ID imports stored null even for team-owned projects.
			prior.TeamID = toTeamID(tt.requestTeam)
			if diags := importResp.State.Set(ctx, prior); diags.HasError() {
				t.Fatalf("set prior state diagnostics: %v", diags)
			}
			readResp := &resource.ReadResponse{State: importResp.State}
			res.Read(ctx, resource.ReadRequest{State: importResp.State}, readResp)
			if readResp.Diagnostics.HasError() {
				t.Fatalf("read diagnostics: %v", readResp.Diagnostics)
			}
			assertProjectOwnership(t, readResp.State, tt.wantTeam)
		})
	}
}

func assertProjectOwnership(t *testing.T, state tfsdk.State, wantTeam string) {
	t.Helper()
	var project Project
	if diags := state.Get(context.Background(), &project); diags.HasError() {
		t.Fatalf("get state diagnostics: %v", diags)
	}
	if project.ID.ValueString() != "prj_123" {
		t.Errorf("id = %q, want prj_123", project.ID.ValueString())
	}
	want := types.StringNull()
	if wantTeam != "" {
		want = types.StringValue(wantTeam)
	}
	if !project.TeamID.Equal(want) {
		t.Errorf("team_id = %s, want %s", project.TeamID, want)
	}
}
