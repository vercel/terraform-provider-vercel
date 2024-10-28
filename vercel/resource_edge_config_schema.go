package vercel

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v2/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &edgeConfigSchemaResource{}
	_ resource.ResourceWithConfigure   = &edgeConfigSchemaResource{}
	_ resource.ResourceWithImportState = &edgeConfigSchemaResource{}
)

func newEdgeConfigSchemaResource() resource.Resource {
	return &edgeConfigSchemaResource{}
}

type edgeConfigSchemaResource struct {
	client *client.Client
}

func (r *edgeConfigSchemaResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_config_schema"
}

func (r *edgeConfigSchemaResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Schema returns the schema information for an edgeConfig resource.
func (r *edgeConfigSchemaResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
An Edge Config Schema provides an existing Edge Config with a JSON schema. Use schema protection to prevent unexpected updates that may cause bugs or downtime.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the Edge Config that the schema should apply to.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"definition": schema.StringAttribute{
				Required:    true,
				Description: "A JSON schema that will be used to validate data in the Edge Config.",
				Validators:  []validator.String{validateJSON()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Edge Config should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured()},
			},
		},
	}
}

type EdgeConfigSchema struct {
	ID         types.String `tfsdk:"id"`
	Definition types.String `tfsdk:"definition"`
	TeamID     types.String `tfsdk:"team_id"`
}

func (e EdgeConfigSchema) JSONDefinition() (i any, err error) {
	err = json.Unmarshal([]byte(e.Definition.ValueString()), &i)
	return i, err
}

func responseToEdgeConfigSchema(out client.EdgeConfigSchema, def types.String) EdgeConfigSchema {
	return EdgeConfigSchema{
		ID:         types.StringValue(out.ID),
		Definition: def,
		TeamID:     toTeamID(out.TeamID),
	}
}

// Create will create an edgeConfigSchema within Vercel.
// This is called automatically by the provider when a new resource should be created.
func (r *edgeConfigSchemaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EdgeConfigSchema
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	definition, err := plan.JSONDefinition()
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid value provided",
			"`definition` must be a valid JSON document, but it could not be parsed: %s."+err.Error(),
		)
		return
	}

	out, err := r.client.UpsertEdgeConfigSchema(ctx, client.EdgeConfigSchema{
		ID:         plan.ID.ValueString(),
		Definition: definition,
		TeamID:     plan.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Edge Config Schema",
			"Could not create Edge Config Schema, unexpected error: "+err.Error(),
		)
		return
	}

	result := responseToEdgeConfigSchema(out, plan.Definition)
	tflog.Info(ctx, "created Edge Config Schema", map[string]interface{}{
		"team_id":        plan.TeamID.ValueString(),
		"edge_config_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read edgeConfigSchema information by requesting it from the Vercel API, and will update terraform
// with this information.
func (r *edgeConfigSchemaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EdgeConfigSchema
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetEdgeConfigSchema(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Edge Config Schema",
			fmt.Sprintf("Could not get Edge Config Schema %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := responseToEdgeConfigSchema(out, state.Definition)
	tflog.Info(ctx, "read edge config schema", map[string]interface{}{
		"team_id":        result.TeamID.ValueString(),
		"edge_config_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update does nothing.
func (r *edgeConfigSchemaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EdgeConfigSchema
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	definition, err := plan.JSONDefinition()
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid value provided",
			"`definition` must be a valid JSON document, but it could not be parsed: %s."+err.Error(),
		)
		return
	}
	out, err := r.client.UpsertEdgeConfigSchema(ctx, client.EdgeConfigSchema{
		ID:         plan.ID.ValueString(),
		Definition: definition,
		TeamID:     plan.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Edge Config Schema",
			"Could not create Edge Config Schema, unexpected error: "+err.Error(),
		)
		return
	}

	result := responseToEdgeConfigSchema(out, plan.Definition)
	tflog.Info(ctx, "created Edge Config Schema", map[string]interface{}{
		"team_id":        plan.TeamID.ValueString(),
		"edge_config_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes an Edge Config Schema.
func (r *edgeConfigSchemaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EdgeConfigSchema
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteEdgeConfigSchema(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Edge Config Schema",
			fmt.Sprintf(
				"Could not delete Edge Config Schema %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted edge config schema", map[string]interface{}{
		"team_id":        state.TeamID.ValueString(),
		"edge_config_id": state.ID.ValueString(),
	})
}

func (r *edgeConfigSchemaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, id, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Edge Config Schema",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/edge_config_id\" or \"edge_config_id\"", req.ID),
		)
	}

	out, err := r.client.GetEdgeConfigSchema(ctx, id, teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Edge Config Schema",
			fmt.Sprintf("Could not get Edge Config Schema %s %s, unexpected error: %s",
				teamID,
				id,
				err,
			),
		)
		return
	}

	def, err := json.Marshal(out.Definition)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Edge Config Schema",
			fmt.Sprintf("Could not marshal Edge Config Schema %s %s, unexpected error: %s",
				teamID, id, err,
			),
		)
		return
	}
	result := responseToEdgeConfigSchema(out, types.StringValue(string(def)))
	tflog.Info(ctx, "import edge config schema", map[string]interface{}{
		"team_id":        result.TeamID.ValueString(),
		"edge_config_id": result.ID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
