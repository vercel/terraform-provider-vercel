package vercel

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
