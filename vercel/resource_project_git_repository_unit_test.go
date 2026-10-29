package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

const (
	gitRelinkOldRepo = "vercel/old-repo"
	gitRelinkNewRepo = "vercel/new-repo"
)

type gitRelinkServer struct {
	// linkStatuses maps a repo to the statuses returned for successive link attempts.
	// A status of 0 links the repo successfully.
	linkStatuses map[string][]int
	linkAttempts map[string]int
	unlinks      int
	branches     []string
	linked       string
}

func (s *gitRelinkServer) start(t *testing.T) *httptest.Server {
	t.Helper()
	s.linkAttempts = map[string]int{}
	s.linked = gitRelinkOldRepo
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v10/projects/prj_123":
		case req.Method == http.MethodPatch && req.URL.Path == "/v9/projects/prj_123":
		case req.Method == http.MethodDelete && req.URL.Path == "/v9/projects/prj_123/link":
			s.unlinks++
			s.linked = ""
		case req.Method == http.MethodPost && req.URL.Path == "/v9/projects/prj_123/link":
			var body struct {
				Repo string `json:"repo"`
				Type string `json:"type"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Errorf("decoding link request: %s", err)
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			attempt := s.linkAttempts[body.Repo]
			s.linkAttempts[body.Repo] = attempt + 1
			statuses := s.linkStatuses[body.Repo]
			if attempt < len(statuses) && statuses[attempt] != 0 {
				w.WriteHeader(statuses[attempt])
				fmt.Fprintf(w, `{"error":{"code":"failed","message":"link failed with %d"}}`, statuses[attempt])
				return
			}
			s.linked = body.Repo
		case req.Method == http.MethodPatch && req.URL.Path == "/v9/projects/prj_123/branch":
			var body struct {
				Branch string `json:"branch"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Errorf("decoding branch request: %s", err)
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			s.branches = append(s.branches, body.Branch)
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL)
			http.NotFound(w, req)
			return
		}
		link := "null"
		if s.linked != "" {
			org, repo, _ := strings.Cut(s.linked, "/")
			link = fmt.Sprintf(`{"type":"github","org":%q,"repo":%q,"productionBranch":"main"}`, org, repo)
		}
		fmt.Fprintf(w, `{"id":"prj_123","name":"example","accountId":"team_123","nodeVersion":"22.x","link":%s}`, link)
	}))
	t.Cleanup(server.Close)
	return server
}

// gitRelinkStateAndPlan imports a project linked to gitRelinkOldRepo and returns
// that state alongside a plan that links gitRelinkNewRepo instead.
func gitRelinkStateAndPlan(t *testing.T, r *projectResource, projectSchema schema.Schema) (tfsdk.State, tfsdk.Plan) {
	t.Helper()
	ctx := context.Background()
	importResp := resource.ImportStateResponse{State: tfsdk.State{Schema: projectSchema}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "team_123/prj_123"}, &importResp)
	if importResp.Diagnostics.HasError() {
		t.Fatal(importResp.Diagnostics)
	}
	plan := tfsdk.Plan{Schema: projectSchema, Raw: importResp.State.Raw}
	if diags := plan.SetAttribute(ctx, path.Root("git_repository"), types.ObjectValueMust(gitRepositoryAttrType.AttrTypes, map[string]attr.Value{
		"type":              types.StringValue("github"),
		"repo":              types.StringValue(gitRelinkNewRepo),
		"production_branch": types.StringValue("main"),
		"deploy_hooks":      types.SetNull(deployHookType),
	})); diags.HasError() {
		t.Fatal(diags)
	}
	return importResp.State, plan
}

func gitRelinkRepo(t *testing.T, state tfsdk.State) types.Object {
	t.Helper()
	var repo types.Object
	if diags := state.GetAttribute(context.Background(), path.Root("git_repository"), &repo); diags.HasError() {
		t.Fatal(diags)
	}
	return repo
}

func TestProjectUpdateRetriesTransientGitRepoLink(t *testing.T) {
	gitRepoLinkRetryDelay = time.Millisecond
	t.Cleanup(func() { gitRepoLinkRetryDelay = 2 * time.Second })

	api := &gitRelinkServer{linkStatuses: map[string][]int{
		gitRelinkNewRepo: {http.StatusConflict, http.StatusConflict},
	}}
	server := api.start(t)

	ctx := context.Background()
	r := &projectResource{client: client.New("test").WithBaseURL(server.URL)}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	state, plan := gitRelinkStateAndPlan(t, r, schemaResp.Schema)

	updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: state.Raw}}
	r.Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("transient link errors were not retried: %v", updateResp.Diagnostics)
	}
	if api.linkAttempts[gitRelinkNewRepo] != 3 {
		t.Errorf("link attempts = %d, want 3", api.linkAttempts[gitRelinkNewRepo])
	}
	if api.linked != gitRelinkNewRepo {
		t.Errorf("linked repo = %q, want %q", api.linked, gitRelinkNewRepo)
	}
	if got := gitRelinkRepo(t, updateResp.State).Attributes()["repo"]; !got.Equal(types.StringValue(gitRelinkNewRepo)) {
		t.Errorf("state repo = %s, want %q", got, gitRelinkNewRepo)
	}
}

func TestProjectUpdateDoesNotRetryPermanentGitRepoLinkErrors(t *testing.T) {
	gitRepoLinkRetryDelay = time.Millisecond
	t.Cleanup(func() { gitRepoLinkRetryDelay = 2 * time.Second })

	api := &gitRelinkServer{linkStatuses: map[string][]int{
		gitRelinkNewRepo: {http.StatusForbidden, http.StatusForbidden, http.StatusForbidden},
	}}
	server := api.start(t)

	ctx := context.Background()
	r := &projectResource{client: client.New("test").WithBaseURL(server.URL)}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	state, plan := gitRelinkStateAndPlan(t, r, schemaResp.Schema)

	updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: state.Raw}}
	r.Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, &updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("permanent link failure did not produce an error")
	}
	if api.linkAttempts[gitRelinkNewRepo] != 1 {
		t.Errorf("link attempts = %d, want 1 for a permission error", api.linkAttempts[gitRelinkNewRepo])
	}
}

func TestProjectUpdateRollsBackFailedGitRepoLink(t *testing.T) {
	gitRepoLinkRetryDelay = time.Millisecond
	t.Cleanup(func() { gitRepoLinkRetryDelay = 2 * time.Second })

	for _, tc := range []struct {
		name            string
		oldRepoStatuses []int
		wantDetail      string
		wantLinked      string
		wantStateRepo   types.Object
	}{
		{
			name:       "rollback succeeds",
			wantDetail: fmt.Sprintf("The previous git repo %s was restored.", gitRelinkOldRepo),
			wantLinked: gitRelinkOldRepo,
			wantStateRepo: types.ObjectValueMust(gitRepositoryAttrType.AttrTypes, map[string]attr.Value{
				"type":              types.StringValue("github"),
				"repo":              types.StringValue(gitRelinkOldRepo),
				"production_branch": types.StringValue("main"),
				"deploy_hooks":      types.SetNull(deployHookType),
			}),
		},
		{
			name:            "rollback fails",
			oldRepoStatuses: []int{http.StatusForbidden},
			wantDetail:      fmt.Sprintf("The previous git repo %s could not be restored either", gitRelinkOldRepo),
			wantLinked:      "",
			wantStateRepo:   types.ObjectNull(gitRepositoryAttrType.AttrTypes),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := &gitRelinkServer{linkStatuses: map[string][]int{
				gitRelinkNewRepo: {http.StatusConflict, http.StatusConflict, http.StatusConflict},
				gitRelinkOldRepo: tc.oldRepoStatuses,
			}}
			server := api.start(t)

			ctx := context.Background()
			r := &projectResource{client: client.New("test").WithBaseURL(server.URL)}
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			state, plan := gitRelinkStateAndPlan(t, r, schemaResp.Schema)

			updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: state.Raw}}
			r.Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, &updateResp)
			if !updateResp.Diagnostics.HasError() {
				t.Fatal("failed link did not produce an error")
			}
			if api.linkAttempts[gitRelinkNewRepo] != gitRepoLinkAttempts {
				t.Errorf("link attempts = %d, want %d", api.linkAttempts[gitRelinkNewRepo], gitRepoLinkAttempts)
			}
			detail := updateResp.Diagnostics.Errors()[0].Detail()
			if !strings.Contains(detail, tc.wantDetail) {
				t.Errorf("error detail = %q, want it to contain %q", detail, tc.wantDetail)
			}
			if api.linked != tc.wantLinked {
				t.Errorf("linked repo = %q, want %q", api.linked, tc.wantLinked)
			}
			if got := gitRelinkRepo(t, updateResp.State); !got.Equal(tc.wantStateRepo) {
				t.Errorf("state git_repository = %s, want %s", got, tc.wantStateRepo)
			}
			if tc.wantLinked == gitRelinkOldRepo && len(api.branches) != 1 {
				t.Errorf("production branch updates = %d, want 1 to restore the previous branch", len(api.branches))
			}
		})
	}
}

func TestTransientAPIErrors(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   bool
	}{
		{status: 409, want: true},
		{status: 429, want: true},
		{status: 500, want: true},
		{status: 502, want: true},
		{status: 400, want: false},
		{status: 403, want: false},
		{status: 404, want: false},
	} {
		t.Run(fmt.Sprint(tc.status), func(t *testing.T) {
			err := fmt.Errorf("error linking git repo: %w", client.APIError{StatusCode: tc.status})
			if got := client.Transient(err); got != tc.want {
				t.Errorf("Transient(%d) = %t, want %t", tc.status, got, tc.want)
			}
		})
	}
	if client.Transient(nil) {
		t.Error("Transient(nil) = true, want false")
	}
	if client.Transient(fmt.Errorf("network trouble")) {
		t.Error("Transient(non-API error) = true, want false")
	}
}
