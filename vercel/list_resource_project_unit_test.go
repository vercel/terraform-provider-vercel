package vercel

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func projectListTestRequest(t *testing.T, r *projectListResource, config projectSearchConfig, limit int64, include bool) list.ListRequest {
	t.Helper()
	ctx := context.Background()
	var configSchema list.ListResourceSchemaResponse
	r.ListResourceConfigSchema(ctx, list.ListResourceSchemaRequest{}, &configSchema)
	rawConfig := tfsdk.State{Schema: configSchema.Schema}
	if diags := rawConfig.Set(ctx, config); diags.HasError() {
		t.Fatal(diags)
	}
	var resourceSchema resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &resourceSchema)
	var identitySchema resource.IdentitySchemaResponse
	r.IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &identitySchema)
	return list.ListRequest{
		Config: tfsdk.Config{Schema: configSchema.Schema, Raw: rawConfig.Raw}, Limit: limit, IncludeResource: include,
		ResourceSchema: resourceSchema.Schema, ResourceIdentitySchema: identitySchema.IdentitySchema,
	}
}

func TestProjectListPaginationFiltersAndLimit(t *testing.T) {
	for _, stop := range []bool{false, true} {
		t.Run(fmt.Sprintf("stop_consumer_%t", stop), func(t *testing.T) {
			pages := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet || req.URL.Path != "/v10/projects" {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL)
				}
				if req.URL.Query().Get("teamId") != "team_default" || req.URL.Query().Get("repoUrl") != "https://github.com/example/web" {
					t.Errorf("query = %s", req.URL.RawQuery)
				}
				pages++
				if pages == 1 {
					fmt.Fprint(w, `{"projects":[{"id":"prj_skip","framework":"astro"}],"pagination":{"next":123}}`)
				} else if pages == 2 {
					if req.URL.Query().Get("until") != "123" {
						t.Errorf("until = %q", req.URL.Query().Get("until"))
					}
					fmt.Fprint(w, `{"projects":[{"id":"prj_1","name":"first","framework":"nextjs"},{"id":"prj_1","name":"first","framework":"nextjs"},{"id":"prj_2","name":"second","framework":"nextjs"}],"pagination":{"next":456}}`)
				} else {
					t.Error("fetched after limit or consumer stop")
					fmt.Fprint(w, `{"projects":[],"pagination":{"next":null}}`)
				}
			}))
			defer server.Close()
			r := &projectListResource{projectResource: projectResource{client: client.New("test").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_default"})}}
			req := projectListTestRequest(t, r, projectSearchConfig{Framework: types.StringValue("nextjs"), RepositoryURL: types.StringValue("https://github.com/example/web")}, 2, false)
			var resp list.ListResultsStream
			r.List(context.Background(), req, &resp)
			var ids []string
			for result := range resp.Results {
				if result.Diagnostics.HasError() {
					t.Fatal(result.Diagnostics)
				}
				var identity projectIdentity
				if diags := result.Identity.Get(context.Background(), &identity); diags.HasError() {
					t.Fatal(diags)
				}
				if identity.TeamID.ValueString() != "team_default" || result.DisplayName == "" {
					t.Fatalf("identity = %#v, name = %s", identity, result.DisplayName)
				}
				if !result.Resource.Raw.IsNull() {
					t.Fatal("identity-only query populated resource")
				}
				ids = append(ids, identity.ProjectID.ValueString())
				if stop {
					break
				}
			}
			expected := 2
			if stop {
				expected = 1
			}
			if len(ids) != expected || pages != 2 || ids[0] != "prj_1" {
				t.Fatalf("ids = %#v, pages = %d", ids, pages)
			}
		})
	}
}

func TestProjectListFullResourceAndErrors(t *testing.T) {
	for _, failure := range []string{"", "page", "detail"} {
		t.Run("failure_"+failure, func(t *testing.T) {
			pages, details := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet {
					t.Errorf("unexpected mutation: %s", req.Method)
				}
				if req.URL.Query().Get("teamId") != "team_override" {
					t.Errorf("teamId = %q", req.URL.Query().Get("teamId"))
				}
				switch req.URL.Path {
				case "/v10/projects":
					pages++
					if failure == "page" {
						http.Error(w, `{"error":{"code":"bad_request","message":"page failed"}}`, http.StatusBadRequest)
						return
					}
					fmt.Fprint(w, `{"projects":[{"id":"prj_1","name":"web","accountId":"team_override"}],"pagination":{"next":null}}`)
				case "/v10/projects/prj_1":
					details++
					if failure == "detail" {
						http.Error(w, `{"error":{"code":"bad_request","message":"detail failed"}}`, http.StatusBadRequest)
						return
					}
					fmt.Fprint(w, `{"id":"prj_1","name":"web","accountId":"team_override","nodeVersion":"22.x","framework":"nextjs","resourceConfig":{"functionDefaultRegions":["iad1"]},"passport":{"connectorId":"connector_1","deploymentType":"all"}}`)
				default:
					t.Errorf("unexpected request: %s", req.URL)
					http.NotFound(w, req)
				}
			}))
			defer server.Close()
			r := &projectListResource{projectResource: projectResource{client: client.New("test").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_default"})}}
			req := projectListTestRequest(t, r, projectSearchConfig{TeamID: types.StringValue("team_override")}, 100, true)
			var resp list.ListResultsStream
			r.List(context.Background(), req, &resp)
			results := 0
			for result := range resp.Results {
				results++
				if failure != "" {
					if !result.Diagnostics.HasError() {
						t.Fatal("expected diagnostic")
					}
					continue
				}
				if result.Diagnostics.HasError() {
					t.Fatal(result.Diagnostics)
				}
				var model Project
				if diags := result.Resource.Get(context.Background(), &model); diags.HasError() {
					t.Fatal(diags)
				}
				if model.ID.ValueString() != "prj_1" || model.TeamID.ValueString() != "team_override" || model.Framework.ValueString() != "nextjs" {
					t.Fatalf("model = %#v", model)
				}
				if !model.BuildMachineType.IsNull() {
					t.Fatalf("omitted build machine type = %s, want null", model.BuildMachineType)
				}
				if !model.Environment.IsNull() {
					t.Fatal("inline environment should not be read")
				}
			}
			wantDetails := 1
			if failure == "page" {
				wantDetails = 0
			}
			if pages != 1 || details != wantDetails || results != 1 {
				t.Fatalf("pages = %d, details = %d, results = %d", pages, details, results)
			}
		})
	}
}

func TestProjectListCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) { t.Error("request after cancellation") }))
	defer server.Close()
	r := &projectListResource{projectResource: projectResource{client: client.New("test").WithBaseURL(server.URL)}}
	req := projectListTestRequest(t, r, projectSearchConfig{}, 100, false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var resp list.ListResultsStream
	r.List(ctx, req, &resp)
	count := 0
	for result := range resp.Results {
		count++
		if !result.Diagnostics.HasError() {
			t.Fatal("expected cancellation diagnostic")
		}
	}
	if count != 1 {
		t.Fatalf("results = %d", count)
	}
}

func TestProjectListPaginationErrors(t *testing.T) {
	for _, pageError := range []bool{true, false} {
		t.Run(fmt.Sprintf("page_error_%t", pageError), func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls++
				if calls > 1 && pageError {
					http.Error(w, `{"error":{"code":"bad_request","message":"page failed"}}`, http.StatusBadRequest)
					return
				}
				if calls == 1 {
					fmt.Fprint(w, `{"projects":[{"id":"prj_1","name":"first"}],"pagination":{"next":123}}`)
				} else {
					fmt.Fprint(w, `{"projects":[],"pagination":{"next":123}}`)
				}
			}))
			defer server.Close()
			r := &projectListResource{projectResource: projectResource{client: client.New("test").WithBaseURL(server.URL)}}
			req := projectListTestRequest(t, r, projectSearchConfig{}, 10, false)
			var resp list.ListResultsStream
			r.List(context.Background(), req, &resp)
			results, failures := 0, 0
			for result := range resp.Results {
				if result.Diagnostics.HasError() {
					failures++
				} else {
					results++
				}
			}
			if calls != 2 || results != 1 || failures != 1 {
				t.Fatalf("calls = %d, results = %d, failures = %d", calls, results, failures)
			}
		})
	}
}
