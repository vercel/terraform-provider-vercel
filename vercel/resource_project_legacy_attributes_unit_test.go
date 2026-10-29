package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

// The deprecated preview_comments and serverless_function_region attributes are
// mutually exclusive with enable_preview_feedback and
// resource_config.function_default_regions, but every attribute of both pairs is
// Computed. Terraform marks unconfigured Computed attributes as "known after
// apply" whenever anything else in the resource changes, so these tests plan a
// project change through the provider server and assert that the configured half
// of each pair decides the planned value for both halves, and that the planned
// values survive the update request and the readback unchanged.

func projectTestSchema(t *testing.T) (context.Context, resourceschema.Schema) {
	t.Helper()
	ctx := context.Background()
	resp := &resource.SchemaResponse{}
	newProjectResource().Schema(ctx, resource.SchemaRequest{}, resp)
	return ctx, resp.Schema
}

func projectObjectType(t *testing.T, ctx context.Context, schema resourceschema.Schema) tftypes.Object {
	t.Helper()
	objectType, ok := schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("project schema type = %T, want tftypes.Object", schema.Type().TerraformType(ctx))
	}
	return objectType
}

func nullProjectValue(t *testing.T, ctx context.Context, schema resourceschema.Schema) tftypes.Value {
	t.Helper()
	objectType := projectObjectType(t, ctx, schema)
	attributes := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, attributeType := range objectType.AttributeTypes {
		attributes[name] = tftypes.NewValue(attributeType, nil)
	}
	return tftypes.NewValue(objectType, attributes)
}

func withProjectAttributes(t *testing.T, ctx context.Context, schema resourceschema.Schema, base tftypes.Value, values map[string]tftypes.Value) tftypes.Value {
	t.Helper()
	var attributes map[string]tftypes.Value
	if err := base.As(&attributes); err != nil {
		t.Fatalf("decoding project value: %s", err)
	}
	for name, value := range values {
		attributes[name] = value
	}
	return tftypes.NewValue(projectObjectType(t, ctx, schema), attributes)
}

func projectFromValue(t *testing.T, ctx context.Context, schema resourceschema.Schema, raw tftypes.Value) Project {
	t.Helper()
	state := tfsdk.State{Schema: schema, Raw: raw}
	var project Project
	diags := state.Get(ctx, &project)
	if diags.HasError() {
		t.Fatalf("state.Get() returned diagnostics: %v", diags)
	}
	return project
}

func projectValue(t *testing.T, ctx context.Context, schema resourceschema.Schema, project Project) tftypes.Value {
	t.Helper()
	state := tfsdk.State{
		Schema: schema,
		Raw:    tftypes.NewValue(schema.Type().TerraformType(ctx), nil),
	}
	diags := state.Set(ctx, project)
	if diags.HasError() {
		t.Fatalf("state.Set() returned diagnostics: %v", diags)
	}
	return state.Raw
}

// projectProposedNewState mirrors the proposed new state Terraform core sends
// with a plan request: a value that is null in the configuration keeps the prior
// state value when the attribute is Computed, anything else comes from the
// configuration.
func projectProposedNewState(t *testing.T, ctx context.Context, schema resourceschema.Schema, prior, config tftypes.Value) tftypes.Value {
	t.Helper()
	proposed, err := tftypes.Transform(config, func(p *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
		if len(p.Steps()) == 0 || !v.IsNull() {
			return v, nil
		}
		attribute, err := schema.AttributeAtTerraformPath(ctx, p)
		if err != nil || !attribute.IsComputed() {
			return v, nil
		}
		priorValue, _, err := tftypes.WalkAttributePath(prior, p)
		if err != nil {
			return v, nil
		}
		priorAttribute, ok := priorValue.(tftypes.Value)
		if !ok {
			return v, nil
		}
		return priorAttribute, nil
	})
	if err != nil {
		t.Fatalf("building proposed new state: %s", err)
	}
	return proposed
}

func planProjectChange(t *testing.T, ctx context.Context, schema resourceschema.Schema, prior, config tftypes.Value) Project {
	t.Helper()
	proposed := projectProposedNewState(t, ctx, schema, prior, config)

	server := providerserver.NewProtocol6(New())()
	resp, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName:         "vercel_project",
		PriorState:       projectDynamicValue(t, ctx, schema, prior),
		Config:           projectDynamicValue(t, ctx, schema, config),
		ProposedNewState: projectDynamicValue(t, ctx, schema, proposed),
	})
	if err != nil {
		t.Fatalf("PlanResourceChange() returned error: %s", err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("PlanResourceChange() diagnostic: %s: %s", d.Summary, d.Detail)
		}
	}
	planned, err := resp.PlannedState.Unmarshal(schema.Type().TerraformType(ctx))
	if err != nil {
		t.Fatalf("unmarshalling planned state: %s", err)
	}
	return projectFromValue(t, ctx, schema, planned)
}

func projectDynamicValue(t *testing.T, ctx context.Context, schema resourceschema.Schema, value tftypes.Value) *tfprotov6.DynamicValue {
	t.Helper()
	dynamicValue, err := tfprotov6.NewDynamicValue(schema.Type().TerraformType(ctx), value)
	if err != nil {
		t.Fatalf("building dynamic value: %s", err)
	}
	return &dynamicValue
}

// projectUpdateResponse is the project the Vercel API returns after applying an
// update request, limited to the fields the deprecated attributes feed into.
func projectUpdateResponse(t *testing.T, ctx context.Context, plan Project) client.ProjectResponse {
	t.Helper()
	req, diags := plan.toUpdateProjectRequest(ctx, "example")
	if diags.HasError() {
		t.Fatalf("toUpdateProjectRequest() returned diagnostics: %v", diags)
	}
	response := client.ProjectResponse{
		ID:                    "prj_123",
		Name:                  "example",
		EnablePreviewFeedback: req.EnablePreviewFeedback,
	}
	if req.ResourceConfig != nil {
		if len(req.ResourceConfig.FunctionDefaultRegions) > 0 {
			region := req.ResourceConfig.FunctionDefaultRegions[0]
			response.ServerlessFunctionRegion = &region
		}
		response.ResourceConfig = &client.ResourceConfigResponse{
			FunctionDefaultMemoryType: req.ResourceConfig.FunctionDefaultMemoryType,
			FunctionDefaultTimeout:    req.ResourceConfig.FunctionDefaultTimeout,
			FunctionDefaultRegions:    req.ResourceConfig.FunctionDefaultRegions,
		}
		if req.ResourceConfig.Fluid != nil {
			response.ResourceConfig.Fluid = *req.ResourceConfig.Fluid
		}
	}
	return response
}

func projectFunctionDefaultRegions(t *testing.T, ctx context.Context, project Project) []string {
	t.Helper()
	resourceConfig, diags := project.resourceConfig(ctx)
	if diags.HasError() {
		t.Fatalf("resourceConfig() returned diagnostics: %v", diags)
	}
	if resourceConfig == nil {
		return nil
	}
	if resourceConfig.FunctionDefaultRegions.IsUnknown() {
		t.Fatal("function_default_regions = (known after apply), want a known set of regions")
	}
	var regions []string
	diags = resourceConfig.FunctionDefaultRegions.ElementsAs(ctx, &regions, false)
	if diags.HasError() {
		t.Fatalf("FunctionDefaultRegions.ElementsAs() returned diagnostics: %v", diags)
	}
	return regions
}

func TestProjectPlanMirrorsConfiguredPreviewCommentsIntoPreviewFeedback(t *testing.T) {
	ctx, schema := projectTestSchema(t)
	enabled := true

	priorConfig := withProjectAttributes(t, ctx, schema, nullProjectValue(t, ctx, schema), map[string]tftypes.Value{
		"name":             tftypes.NewValue(tftypes.String, "example"),
		"preview_comments": tftypes.NewValue(tftypes.Bool, true),
	})
	prior, err := convertResponseToProject(ctx, client.ProjectResponse{
		ID:                    "prj_123",
		Name:                  "example",
		EnablePreviewFeedback: &enabled,
	}, projectFromValue(t, ctx, schema, priorConfig), nil)
	if err != nil {
		t.Fatalf("convertResponseToProject() returned error: %v", err)
	}

	// Flipping the deprecated attribute must not leave its modern counterpart
	// unknown, and must not send the stale prior value to the API.
	config := withProjectAttributes(t, ctx, schema, nullProjectValue(t, ctx, schema), map[string]tftypes.Value{
		"name":             tftypes.NewValue(tftypes.String, "example"),
		"preview_comments": tftypes.NewValue(tftypes.Bool, false),
	})
	planned := planProjectChange(t, ctx, schema, projectValue(t, ctx, schema, prior), config)

	if planned.EnablePreviewFeedback.IsUnknown() {
		t.Fatal("planned enable_preview_feedback = (known after apply), want false")
	}
	if planned.EnablePreviewFeedback.ValueBool() || planned.PreviewComments.ValueBool() {
		t.Fatalf("planned preview_comments = %v, enable_preview_feedback = %v, want false for both", planned.PreviewComments, planned.EnablePreviewFeedback)
	}

	readback, err := convertResponseToProject(ctx, projectUpdateResponse(t, ctx, planned), planned, nil)
	if err != nil {
		t.Fatalf("convertResponseToProject() returned error: %v", err)
	}
	if !readback.PreviewComments.Equal(planned.PreviewComments) || !readback.EnablePreviewFeedback.Equal(planned.EnablePreviewFeedback) {
		t.Fatalf("readback preview_comments = %v, enable_preview_feedback = %v, want the planned false", readback.PreviewComments, readback.EnablePreviewFeedback)
	}
}

func TestProjectPlanMirrorsConfiguredPreviewFeedbackIntoPreviewComments(t *testing.T) {
	ctx, schema := projectTestSchema(t)
	enabled := true

	priorConfig := withProjectAttributes(t, ctx, schema, nullProjectValue(t, ctx, schema), map[string]tftypes.Value{
		"name":                    tftypes.NewValue(tftypes.String, "example"),
		"enable_preview_feedback": tftypes.NewValue(tftypes.Bool, true),
	})
	prior, err := convertResponseToProject(ctx, client.ProjectResponse{
		ID:                    "prj_123",
		Name:                  "example",
		EnablePreviewFeedback: &enabled,
	}, projectFromValue(t, ctx, schema, priorConfig), nil)
	if err != nil {
		t.Fatalf("convertResponseToProject() returned error: %v", err)
	}

	// An unrelated change must not turn the deprecated attribute into "known
	// after apply".
	config := withProjectAttributes(t, ctx, schema, priorConfig, map[string]tftypes.Value{
		"build_command": tftypes.NewValue(tftypes.String, "npm run build"),
	})
	planned := planProjectChange(t, ctx, schema, projectValue(t, ctx, schema, prior), config)

	if planned.PreviewComments.IsUnknown() {
		t.Fatal("planned preview_comments = (known after apply), want true")
	}
	if !planned.PreviewComments.ValueBool() || !planned.EnablePreviewFeedback.ValueBool() {
		t.Fatalf("planned preview_comments = %v, enable_preview_feedback = %v, want true for both", planned.PreviewComments, planned.EnablePreviewFeedback)
	}
}

func TestProjectPlanMirrorsConfiguredServerlessFunctionRegionIntoResourceConfig(t *testing.T) {
	ctx, schema := projectTestSchema(t)
	region := "iad1"

	priorConfig := withProjectAttributes(t, ctx, schema, nullProjectValue(t, ctx, schema), map[string]tftypes.Value{
		"name":                       tftypes.NewValue(tftypes.String, "example"),
		"serverless_function_region": tftypes.NewValue(tftypes.String, region),
	})
	prior, err := convertResponseToProject(ctx, client.ProjectResponse{
		ID:                       "prj_123",
		Name:                     "example",
		ServerlessFunctionRegion: &region,
		ResourceConfig: &client.ResourceConfigResponse{
			FunctionDefaultRegions: []string{region},
		},
	}, projectFromValue(t, ctx, schema, priorConfig), nil)
	if err != nil {
		t.Fatalf("convertResponseToProject() returned error: %v", err)
	}

	// An unrelated change must not turn the whole resource config into "known
	// after apply" just because the deprecated region is used instead.
	unrelated := withProjectAttributes(t, ctx, schema, priorConfig, map[string]tftypes.Value{
		"build_command": tftypes.NewValue(tftypes.String, "npm run build"),
	})
	planned := planProjectChange(t, ctx, schema, projectValue(t, ctx, schema, prior), unrelated)

	if planned.ResourceConfig.IsUnknown() {
		t.Fatal("planned resource_config = (known after apply), want the regions implied by serverless_function_region")
	}
	if got := projectFunctionDefaultRegions(t, ctx, planned); len(got) != 1 || got[0] != region {
		t.Fatalf("planned function_default_regions = %v, want [%s]", got, region)
	}

	// Changing the deprecated region must plan and send the new region rather
	// than the regions already in state.
	moved := withProjectAttributes(t, ctx, schema, nullProjectValue(t, ctx, schema), map[string]tftypes.Value{
		"name":                       tftypes.NewValue(tftypes.String, "example"),
		"serverless_function_region": tftypes.NewValue(tftypes.String, "sfo1"),
	})
	planned = planProjectChange(t, ctx, schema, projectValue(t, ctx, schema, prior), moved)

	if got := projectFunctionDefaultRegions(t, ctx, planned); len(got) != 1 || got[0] != "sfo1" {
		t.Fatalf("planned function_default_regions = %v, want [sfo1]", got)
	}

	readback, err := convertResponseToProject(ctx, projectUpdateResponse(t, ctx, planned), planned, nil)
	if err != nil {
		t.Fatalf("convertResponseToProject() returned error: %v", err)
	}
	if got := readback.ServerlessFunctionRegion.ValueString(); got != "sfo1" {
		t.Fatalf("readback serverless_function_region = %q, want sfo1", got)
	}
	if got := projectFunctionDefaultRegions(t, ctx, readback); len(got) != 1 || got[0] != "sfo1" {
		t.Fatalf("readback function_default_regions = %v, want [sfo1]", got)
	}
}

func TestProjectPlanMirrorsServerlessFunctionRegionIntoConfiguredResourceConfig(t *testing.T) {
	ctx, schema := projectTestSchema(t)
	region := "iad1"
	cpuType := "standard"

	resourceConfigType := projectObjectType(t, ctx, schema).AttributeTypes["resource_config"].(tftypes.Object)
	resourceConfigValue := tftypes.NewValue(resourceConfigType, map[string]tftypes.Value{
		"function_default_cpu_type": tftypes.NewValue(tftypes.String, cpuType),
		"function_default_timeout":  tftypes.NewValue(tftypes.Number, nil),
		"function_default_regions":  tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),
		"fluid":                     tftypes.NewValue(tftypes.Bool, nil),
	})
	priorConfig := withProjectAttributes(t, ctx, schema, nullProjectValue(t, ctx, schema), map[string]tftypes.Value{
		"name":                       tftypes.NewValue(tftypes.String, "example"),
		"serverless_function_region": tftypes.NewValue(tftypes.String, region),
		"resource_config":            resourceConfigValue,
	})
	prior, err := convertResponseToProject(ctx, client.ProjectResponse{
		ID:                       "prj_123",
		Name:                     "example",
		ServerlessFunctionRegion: &region,
		ResourceConfig: &client.ResourceConfigResponse{
			FunctionDefaultMemoryType: &cpuType,
			FunctionDefaultRegions:    []string{region},
		},
	}, projectFromValue(t, ctx, schema, priorConfig), nil)
	if err != nil {
		t.Fatalf("convertResponseToProject() returned error: %v", err)
	}

	// The regions are omitted from a configured resource config block in favour
	// of the deprecated attribute, so they must not become known after apply.
	config := withProjectAttributes(t, ctx, schema, priorConfig, map[string]tftypes.Value{
		"build_command": tftypes.NewValue(tftypes.String, "npm run build"),
	})
	planned := planProjectChange(t, ctx, schema, projectValue(t, ctx, schema, prior), config)

	if got := projectFunctionDefaultRegions(t, ctx, planned); len(got) != 1 || got[0] != region {
		t.Fatalf("planned function_default_regions = %v, want [%s]", got, region)
	}
}

func TestProjectPlanKeepsUnconfiguredServerlessFunctionRegionNull(t *testing.T) {
	ctx, schema := projectTestSchema(t)

	resourceConfigType := projectObjectType(t, ctx, schema).AttributeTypes["resource_config"].(tftypes.Object)
	resourceConfigValue := tftypes.NewValue(resourceConfigType, map[string]tftypes.Value{
		"function_default_cpu_type": tftypes.NewValue(tftypes.String, nil),
		"function_default_timeout":  tftypes.NewValue(tftypes.Number, nil),
		"function_default_regions": tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{
			tftypes.NewValue(tftypes.String, "iad1"),
		}),
		"fluid": tftypes.NewValue(tftypes.Bool, nil),
	})
	priorConfig := withProjectAttributes(t, ctx, schema, nullProjectValue(t, ctx, schema), map[string]tftypes.Value{
		"name":            tftypes.NewValue(tftypes.String, "example"),
		"resource_config": resourceConfigValue,
	})
	prior, err := convertResponseToProject(ctx, client.ProjectResponse{
		ID:   "prj_123",
		Name: "example",
		ResourceConfig: &client.ResourceConfigResponse{
			FunctionDefaultRegions: []string{"iad1"},
		},
	}, projectFromValue(t, ctx, schema, priorConfig), nil)
	if err != nil {
		t.Fatalf("convertResponseToProject() returned error: %v", err)
	}
	if !prior.ServerlessFunctionRegion.IsNull() {
		t.Fatalf("serverless_function_region = %v, want null for unconfigured deprecated attribute", prior.ServerlessFunctionRegion)
	}

	config := withProjectAttributes(t, ctx, schema, priorConfig, map[string]tftypes.Value{
		"build_command": tftypes.NewValue(tftypes.String, "npm run build"),
	})
	planned := planProjectChange(t, ctx, schema, projectValue(t, ctx, schema, prior), config)

	if !planned.ServerlessFunctionRegion.IsNull() {
		t.Fatalf("planned serverless_function_region = %v, want null", planned.ServerlessFunctionRegion)
	}
	if got := projectFunctionDefaultRegions(t, ctx, planned); len(got) != 1 || got[0] != "iad1" {
		t.Fatalf("planned function_default_regions = %v, want [iad1]", got)
	}
}
