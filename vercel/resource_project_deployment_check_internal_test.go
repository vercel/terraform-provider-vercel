package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestProjectDeploymentCheckVercelCISourceToClient(t *testing.T) {
	ctx := context.Background()
	filters := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("vercel-docs")})
	selection := types.ObjectValueMust(projectDeploymentCheckSourceSelectionAttrTypes, map[string]attr.Value{
		"kind":             types.StringValue("turborepo"),
		"job":              types.StringValue("Turborepo"),
		"task":             types.StringValue("e2e:ci"),
		"filters":          filters,
		"fail_if_no_match": types.BoolValue(true),
	})
	source := types.ObjectValueMust(projectDeploymentCheckSourceAttrTypes, map[string]attr.Value{
		"kind":                 types.StringValue("vercel"),
		"sub_kind":             types.StringValue("vercel-ci-check"),
		"origin":               types.StringNull(),
		"selection":            selection,
		"external_check_name":  types.StringNull(),
		"provider":             types.StringNull(),
		"webhook_id":           types.StringNull(),
		"external_resource_id": types.StringNull(),
	})

	got, diags := projectDeploymentCheckSourceToClient(ctx, source)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if got.Kind != "vercel" || got.SubKind != "vercel-ci-check" || got.Origin != "api" || got.Selection == nil {
		t.Fatalf("source = %#v", got)
	}
	if got.Selection.Kind != "turborepo" || got.Selection.Job != "Turborepo" || got.Selection.Task != "e2e:ci" {
		t.Fatalf("selection = %#v", got.Selection)
	}
	if got.Selection.Filters == nil || len(*got.Selection.Filters) != 1 || (*got.Selection.Filters)[0] != "vercel-docs" {
		t.Fatalf("filters = %#v", got.Selection.Filters)
	}
	if got.Selection.FailIfNoMatch == nil || !*got.Selection.FailIfNoMatch {
		t.Fatalf("fail_if_no_match = %#v", got.Selection.FailIfNoMatch)
	}
}
