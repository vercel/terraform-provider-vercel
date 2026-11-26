package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestResponseToProjectSpeedInsights(t *testing.T) {
	result := responseToProjectSpeedInsights(client.ProjectSpeedInsights{
		TeamID:    "team_123",
		ProjectID: "prj_123",
		Enabled:   true,
	})
	if result.ID.ValueString() != "prj_123" || result.ProjectID.ValueString() != "prj_123" || result.TeamID.ValueString() != "team_123" {
		t.Fatalf("result = %#v", result)
	}
}

func TestProjectSpeedInsightsResourceSchema(t *testing.T) {
	response := &resource.SchemaResponse{}
	newProjectSpeedInsightsResource().Schema(context.Background(), resource.SchemaRequest{}, response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics = %v", response.Diagnostics)
	}

	if !response.Schema.Attributes["project_id"].IsRequired() {
		t.Fatal("project_id must be required")
	}
	if _, ok := response.Schema.Attributes["enabled"]; ok {
		t.Fatal("enabled must not be exposed")
	}
}
