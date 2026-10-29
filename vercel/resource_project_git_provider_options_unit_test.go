package vercel

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestProjectGitProviderOptionsToUpdateProjectRequest(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name      string
		overrides map[string]attr.Value
		expected  string
	}{
		{
			name:      "unknown values are omitted",
			overrides: nil,
			expected:  "null",
		},
		{
			name: "null values are omitted",
			overrides: map[string]attr.Value{
				"require_verified_commits":       types.BoolNull(),
				"create_deployments":             types.BoolNull(),
				"repository_dispatch_events":     types.BoolNull(),
				"git_commit_status":              types.BoolNull(),
				"consolidated_git_commit_status": types.ObjectNull(consolidatedGitCommitStatusAttrType.AttrTypes),
			},
			expected: "null",
		},
		{
			name: "require_verified_commits true is sent",
			overrides: map[string]attr.Value{
				"require_verified_commits": types.BoolValue(true),
			},
			expected: `{"requireVerifiedCommits":true}`,
		},
		{
			name: "require_verified_commits false is sent",
			overrides: map[string]attr.Value{
				"require_verified_commits": types.BoolValue(false),
			},
			expected: `{"requireVerifiedCommits":false}`,
		},
		{
			name: "git_commit_status true is sent",
			overrides: map[string]attr.Value{
				"git_commit_status": types.BoolValue(true),
			},
			expected: `{"gitCommitStatus":true}`,
		},
		{
			name: "git_commit_status false is sent",
			overrides: map[string]attr.Value{
				"git_commit_status": types.BoolValue(false),
			},
			expected: `{"gitCommitStatus":false}`,
		},
		{
			name: "create_deployments true is sent",
			overrides: map[string]attr.Value{
				"create_deployments": types.BoolValue(true),
			},
			expected: `{"createDeployments":"enabled"}`,
		},
		{
			name: "create_deployments false is sent",
			overrides: map[string]attr.Value{
				"create_deployments": types.BoolValue(false),
			},
			expected: `{"createDeployments":"disabled"}`,
		},
		{
			name: "repository_dispatch_events true is sent",
			overrides: map[string]attr.Value{
				"repository_dispatch_events": types.BoolValue(true),
			},
			expected: `{"disableRepositoryDispatchEvents":false}`,
		},
		{
			name: "repository_dispatch_events false is sent",
			overrides: map[string]attr.Value{
				"repository_dispatch_events": types.BoolValue(false),
			},
			expected: `{"disableRepositoryDispatchEvents":true}`,
		},
		{
			name: "consolidated_git_commit_status is sent",
			overrides: map[string]attr.Value{
				"consolidated_git_commit_status": types.ObjectValueMust(consolidatedGitCommitStatusAttrType.AttrTypes, map[string]attr.Value{
					"enabled":            types.BoolValue(true),
					"propagate_failures": types.BoolValue(false),
				}),
			},
			expected: `{"consolidatedGitCommitStatus":{"enabled":true,"propagateFailures":false}}`,
		},
		{
			name: "known values are sent alongside unknown ones",
			overrides: map[string]attr.Value{
				"require_verified_commits": types.BoolValue(true),
				"create_deployments":       types.BoolValue(true),
			},
			expected: `{"requireVerifiedCommits":true,"createDeployments":"enabled"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project := projectForUpdateRequestTests()
			project.GitProviderOptions = gitProviderOptionsValue(tc.overrides)

			request, diags := project.toUpdateProjectRequest(ctx, project.Name.ValueString())
			if diags.HasError() {
				t.Fatalf("toUpdateProjectRequest() returned diagnostics: %v", diags)
			}

			serialized, err := json.Marshal(request.GitProviderOptions)
			if err != nil {
				t.Fatalf("marshalling git provider options: %s", err)
			}
			if string(serialized) != tc.expected {
				t.Fatalf("gitProviderOptions = %s, want %s", serialized, tc.expected)
			}
		})
	}
}

func gitProviderOptionsValue(overrides map[string]attr.Value) types.Object {
	attrs := map[string]attr.Value{
		"require_verified_commits":       types.BoolUnknown(),
		"create_deployments":             types.BoolUnknown(),
		"repository_dispatch_events":     types.BoolUnknown(),
		"git_commit_status":              types.BoolUnknown(),
		"consolidated_git_commit_status": types.ObjectUnknown(consolidatedGitCommitStatusAttrType.AttrTypes),
	}
	for name, value := range overrides {
		attrs[name] = value
	}
	return types.ObjectValueMust(gitProviderOptionsAttrType.AttrTypes, attrs)
}
