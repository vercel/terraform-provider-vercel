package vercel

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &edgeConfigResource{}
	_ resource.ResourceWithConfigure   = &edgeConfigResource{}
	_ resource.ResourceWithImportState = &edgeConfigResource{}
)

func newEdgeConfigResource() resource.Resource {
	return &edgeConfigResource{}
}

type edgeConfigResource struct {
	client *client.Client
}

func (r *edgeConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_config"
}

func (r *edgeConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *edgeConfigResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides an Edge Config resource.

An Edge Config is a global data store that enables experimentation with feature flags, A/B testing, critical redirects, and more.`,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name/slug of the Edge Config.",
				Required:    true,
				Validators: []validator.String{
					stringRegex(
						regexp.MustCompile(`^[a-z0-9\_\-]{0,32}$`),
						"The name of an Edge Config can only contain up to 32 alphanumeric lowercase characters, hyphens and underscores.",
					),
				},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Edge Config should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"id": schema.StringAttribute{
				Description:   "The ID of the Edge Config.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

type EdgeConfig struct {
	Name   types.String `tfsdk:"name"`
	ID     types.String `tfsdk:"id"`
	TeamID types.String `tfsdk:"team_id"`
}

func responseToEdgeConfig(out client.EdgeConfig) EdgeConfig {
	return EdgeConfig{
		Name:   types.StringValue(out.Slug),
		ID:     types.StringValue(out.ID),
		TeamID: toTeamID(out.TeamID),
	}
}

// Create will create an edgeConfig within Vercel.
// This is called automatically by the provider when a new resource should be created.
func (r *edgeConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EdgeConfig
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateEdgeConfig(ctx, client.CreateEdgeConfigRequest{
		Name:   plan.Name.ValueString(),
		TeamID: plan.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Edge Config",
			"Could not create Edge Config, unexpected error: "+err.Error(),
		)
		return
	}

	result := responseToEdgeConfig(out)
	tflog.Info(ctx, "created Edge Config", map[string]interface{}{
		"team_id":        plan.TeamID.ValueString(),
		"edge_config_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read edgeConfig information by requesting it from the Vercel API, and will update terraform
// with this information.
func (r *edgeConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EdgeConfig
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetEdgeConfig(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Edge Config",
			fmt.Sprintf("Could not get Edge Config %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := responseToEdgeConfig(out)
	tflog.Info(ctx, "read edge config", map[string]interface{}{
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
func (r *edgeConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EdgeConfig
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateEdgeConfig(ctx, client.UpdateEdgeConfigRequest{
		Slug:   plan.Name.ValueString(),
		TeamID: plan.TeamID.ValueString(),
		ID:     plan.ID.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Edge Config",
			fmt.Sprintf("Could not update Edge Config %s %s, unexpected error: %s",
				plan.TeamID.ValueString(),
				plan.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := responseToEdgeConfig(out)
	tflog.Trace(ctx, "update edge config", map[string]interface{}{
		"team_id":        result.TeamID.ValueString(),
		"edge_config_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes an Edge Config.
func (r *edgeConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EdgeConfig
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteEdgeConfig(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting edgeConfig",
			fmt.Sprintf(
				"Could not delete Edge Config %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted Edge Config", map[string]interface{}{
		"team_id":        state.TeamID.ValueString(),
		"edge_config_id": state.ID.ValueString(),
	})
}

func (r *edgeConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, id, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Edge Config",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/edge_config_id\" or \"edge_config_id\"", req.ID),
		)
	}

	out, err := r.client.GetEdgeConfig(ctx, id, teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Edge Config",
			fmt.Sprintf("Could not get Edge Config %s %s, unexpected error: %s",
				teamID,
				id,
				err,
			),
		)
		return
	}

	result := responseToEdgeConfig(out)
	tflog.Info(ctx, "import edge config", map[string]interface{}{
		"team_id":        result.TeamID.ValueString(),
		"edge_config_id": result.ID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
