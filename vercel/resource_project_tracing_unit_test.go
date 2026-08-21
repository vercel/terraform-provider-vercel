package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestResponseToProjectTracing(t *testing.T) {
	result, diags := responseToProjectTracing(context.Background(), client.ProjectTracing{
		TeamID:    "team_123",
		ProjectID: "prj_123",
		Enabled:   true,
		SamplingRules: []client.TraceDrainSamplingRule{
			{Rate: 0.25, Environment: "production", RequestPath: "/api"},
		},
	}, types.ListNull(traceDrainSamplingRuleAttrType))
	if diags.HasError() {
		t.Fatalf("responseToProjectTracing() diagnostics = %v", diags)
	}
	if result.ID.ValueString() != "prj_123" || result.ProjectID.ValueString() != "prj_123" || result.TeamID.ValueString() != "team_123" || !result.Enabled.ValueBool() {
		t.Fatalf("result = %#v", result)
	}

	var rules []TraceDrainSamplingRule
	diags = result.SamplingRules.ElementsAs(context.Background(), &rules, false)
	if diags.HasError() {
		t.Fatalf("ElementsAs() diagnostics = %v", diags)
	}
	if len(rules) != 1 || rules[0].Rate.ValueFloat64() != 0.25 || rules[0].Environment.ValueString() != "production" || rules[0].RequestPath.ValueString() != "/api" {
		t.Fatalf("rules = %#v", rules)
	}
}

func TestProjectTracingResourceSchema(t *testing.T) {
	response := &resource.SchemaResponse{}
	newProjectTracingResource().Schema(context.Background(), resource.SchemaRequest{}, response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics = %v", response.Diagnostics)
	}

	if !response.Schema.Attributes["project_id"].IsRequired() {
		t.Fatal("project_id must be required")
	}
	if !response.Schema.Attributes["enabled"].IsRequired() {
		t.Fatal("enabled must be required")
	}
	if !response.Schema.Attributes["sampling_rules"].IsOptional() {
		t.Fatal("sampling_rules must be optional")
	}
}
