package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestProjectBooleanRequestPayloads(t *testing.T) {
	updateKeys := []string{
		"autoExposeSystemEnvs", "enablePreviewFeedback", "enableProductionFeedback",
		"enableAffectedProjectsDeployments", "previewDeploymentsDisabled", "autoAssignCustomDomains",
		"gitLFS", "serverlessFunctionZeroConfigFailover", "customerSupportCodeVisibility",
		"protectedSourcemaps", "gitForkProtection", "productionDeploymentsFastLane", "directoryListing",
	}
	createKeys := []string{
		"enablePreviewFeedback", "enableProductionFeedback", "enableAffectedProjectsDeployments", "previewDeploymentsDisabled",
	}
	for _, tc := range []struct {
		name  string
		value types.Bool
		want  string
	}{
		{name: "unknown", value: types.BoolUnknown()},
		{name: "null", value: types.BoolNull()},
		{name: "enabled", value: types.BoolValue(true), want: "true"},
		{name: "disabled", value: types.BoolValue(false), want: "false"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project := projectForUpdateRequestTests()
			project.AutoExposeSystemEnvVars = tc.value
			project.EnablePreviewFeedback = tc.value
			project.EnableProductionFeedback = tc.value
			project.EnableAffectedProjectsDeployments = tc.value
			project.PreviewDeploymentsDisabled = tc.value
			project.AutoAssignCustomDomains = tc.value
			project.GitLFS = tc.value
			project.FunctionFailover = tc.value
			project.CustomerSuccessCodeVisibility = tc.value
			project.ProtectedSourcemaps = tc.value
			project.GitForkProtection = tc.value
			project.PrioritiseProductionBuilds = tc.value
			project.DirectoryListing = tc.value
			project.GitProviderOptions = types.ObjectValueMust(gitProviderOptionsAttrType.AttrTypes, map[string]attr.Value{
				"require_verified_commits":       tc.value,
				"create_deployments":             types.BoolNull(),
				"repository_dispatch_events":     types.BoolNull(),
				"git_commit_status":              tc.value,
				"consolidated_git_commit_status": types.ObjectNull(consolidatedGitCommitStatusAttrType.AttrTypes),
			})

			t.Run("update", func(t *testing.T) {
				request, diags := project.toUpdateProjectRequest(context.Background(), "old-name")
				if diags.HasError() {
					t.Fatal(diags)
				}
				body := projectBooleanRequestJSON(t, request)
				var gitOptions map[string]json.RawMessage
				if err := json.Unmarshal(body["gitProviderOptions"], &gitOptions); err != nil {
					t.Fatal(err)
				}
				for _, key := range []string{"requireVerifiedCommits", "gitCommitStatus"} {
					if got := string(gitOptions[key]); got != tc.want {
						t.Errorf("gitProviderOptions.%s = %q, want %q", key, got, tc.want)
					}
				}
				if string(body["name"]) != `"example"` {
					t.Fatalf("name = %s, want example", body["name"])
				}
				for _, key := range updateKeys {
					if got := string(body[key]); got != tc.want {
						t.Errorf("%s = %q, want %q (empty means omitted)", key, got, tc.want)
					}
				}
			})
			t.Run("create", func(t *testing.T) {
				request, diags := project.toCreateProjectRequest(context.Background(), nil)
				if diags.HasError() {
					t.Fatal(diags)
				}
				body := projectBooleanRequestJSON(t, request)
				for _, key := range createKeys {
					if got := string(body[key]); got != tc.want {
						t.Errorf("%s = %q, want %q (empty means omitted)", key, got, tc.want)
					}
				}
			})
		})
	}
}

func TestProjectAffectedDeploymentsResponse(t *testing.T) {
	for _, planValue := range []types.Bool{types.BoolUnknown(), types.BoolNull(), types.BoolValue(false), types.BoolValue(true)} {
		for _, responseValue := range []types.Bool{types.BoolNull(), types.BoolValue(false), types.BoolValue(true)} {
			t.Run(planValue.String()+"/"+responseValue.String(), func(t *testing.T) {
				plan := projectForReadTests()
				plan.EnableAffectedProjectsDeployments = planValue
				result, err := convertResponseToProject(context.Background(), client.ProjectResponse{
					EnableAffectedProjectsDeployments: responseValue.ValueBoolPointer(),
				}, plan, nil)
				if err != nil {
					t.Fatal(err)
				}
				want := responseValue
				if responseValue.IsNull() && planValue.Equal(types.BoolValue(false)) {
					want = planValue
				}
				if !result.EnableAffectedProjectsDeployments.Equal(want) {
					t.Fatalf("EnableAffectedProjectsDeployments = %s, want %s", result.EnableAffectedProjectsDeployments, want)
				}
			})
		}
	}
}

func TestProjectNullBooleansLifecycle(t *testing.T) {
	name := "example"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
			return
		case http.MethodPost, http.MethodPatch:
			var body map[string]json.RawMessage
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Error(err)
				return
			}
			for _, key := range []string{"enableAffectedProjectsDeployments", "enableProductionFeedback", "enablePreviewFeedback"} {
				if value, ok := body[key]; ok {
					t.Errorf("unexpected write to %s: %s", key, value)
				}
			}
			if value, ok := body["name"]; ok {
				if err := json.Unmarshal(value, &name); err != nil {
					t.Error(err)
				}
			}
		}
		fmt.Fprintf(w, `{"id":"prj_123","name":%q,"accountId":"team_123","nodeVersion":"22.x","autoAssignCustomDomains":true,"gitForkProtection":true,"enableAffectedProjectsDeployments":null,"enableProductionFeedback":null,"enablePreviewFeedback":null}`, name)
	}))
	defer server.Close()
	factories := map[string]func() (tfprotov6.ProviderServer, error){
		"vercel": providerserver.NewProtocol6WithError(&customAlertMockProvider{
			Provider: New(),
			client:   client.New("test").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_123"}),
		}),
	}
	config := func(name string) string {
		return fmt.Sprintf(`resource "vercel_project" "test" {
name = %q
node_version = "22.x"
}`, name)
	}
	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: factories, Steps: []tfresource.TestStep{
		{Config: config("example")},
		{Config: config("renamed")},
	}})
}

func projectBooleanRequestJSON(t *testing.T, request any) map[string]json.RawMessage {
	t.Helper()
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatal(err)
	}
	return body
}
