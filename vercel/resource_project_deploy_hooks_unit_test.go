package vercel

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func projectWithPlannedDeployHooks(hooks types.Set) Project {
	plan := nullProject
	plan.GitRepository = types.ObjectValueMust(gitRepositoryAttrType.AttrTypes, map[string]attr.Value{
		"type":              types.StringValue("github"),
		"repo":              types.StringValue("vercel/next.js"),
		"production_branch": types.StringNull(),
		"deploy_hooks":      hooks,
	})
	return plan
}

func projectResponseWithLink(t *testing.T, link string) client.ProjectResponse {
	t.Helper()
	var response client.ProjectResponse
	if err := json.Unmarshal([]byte(`{"id":"prj_123","name":"example","link":`+link+`}`), &response); err != nil {
		t.Fatalf("unmarshalling project response: %v", err)
	}
	return response
}

func readDeployHooks(ctx context.Context, t *testing.T, project Project) types.Set {
	t.Helper()
	git, diags := project.gitRepository(ctx)
	if diags.HasError() {
		t.Fatalf("gitRepository() returned diagnostics: %v", diags)
	}
	if git == nil {
		t.Fatal("gitRepository() = nil, want a git repository")
	}
	return git.DeployHooks
}

func TestConvertResponseToProjectKeepsPlannedEmptyDeployHooksEmpty(t *testing.T) {
	ctx := context.Background()
	emptyHooks := types.SetValueMust(deployHookType, []attr.Value{})

	for name, link := range map[string]string{
		"hooks omitted": `{"type":"github","org":"vercel","repo":"next.js"}`,
		"hooks empty":   `{"type":"github","org":"vercel","repo":"next.js","deployHooks":[]}`,
	} {
		t.Run(name, func(t *testing.T) {
			result, err := convertResponseToProject(ctx, projectResponseWithLink(t, link), projectWithPlannedDeployHooks(emptyHooks), nil)
			if err != nil {
				t.Fatalf("convertResponseToProject() returned error: %v", err)
			}
			if got := readDeployHooks(ctx, t, result); !got.Equal(emptyHooks) {
				t.Fatalf("deploy_hooks = %s, want an empty set", got)
			}
		})
	}
}

func TestConvertResponseToProjectKeepsUnconfiguredDeployHooksNull(t *testing.T) {
	ctx := context.Background()
	plan := projectWithPlannedDeployHooks(types.SetNull(deployHookType))

	result, err := convertResponseToProject(ctx, projectResponseWithLink(t, `{"type":"github","org":"vercel","repo":"next.js"}`), plan, nil)
	if err != nil {
		t.Fatalf("convertResponseToProject() returned error: %v", err)
	}
	if got := readDeployHooks(ctx, t, result); !got.IsNull() {
		t.Fatalf("deploy_hooks = %s, want null", got)
	}
}

func TestConvertResponseToProjectPreservesDeployHooks(t *testing.T) {
	ctx := context.Background()
	plan := projectWithPlannedDeployHooks(types.SetValueMust(deployHookType, []attr.Value{
		types.ObjectValueMust(deployHookType.AttrTypes, map[string]attr.Value{
			"name": types.StringValue("some-hook"),
			"ref":  types.StringValue("main"),
			"url":  types.StringUnknown(),
			"id":   types.StringUnknown(),
		}),
	}))

	result, err := convertResponseToProject(ctx, projectResponseWithLink(t, `{"type":"github","org":"vercel","repo":"next.js","deployHooks":[{"id":"hook_1","name":"some-hook","ref":"main","url":"https://api.vercel.com/v1/integrations/deploy/prj_123/hook_1"}]}`), plan, nil)
	if err != nil {
		t.Fatalf("convertResponseToProject() returned error: %v", err)
	}

	want := types.SetValueMust(deployHookType, []attr.Value{
		types.ObjectValueMust(deployHookType.AttrTypes, map[string]attr.Value{
			"name": types.StringValue("some-hook"),
			"ref":  types.StringValue("main"),
			"url":  types.StringValue("https://api.vercel.com/v1/integrations/deploy/prj_123/hook_1"),
			"id":   types.StringValue("hook_1"),
		}),
	})
	if got := readDeployHooks(ctx, t, result); !got.Equal(want) {
		t.Fatalf("deploy_hooks = %s, want %s", got, want)
	}
}
