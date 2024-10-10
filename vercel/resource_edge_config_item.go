package vercel

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &edgeConfigItemResource{}
	_ resource.ResourceWithConfigure = &edgeConfigItemResource{}
)

func newEdgeConfigItemResource() resource.Resource {
	return &edgeConfigItemResource{}
}

type edgeConfigItemResource struct {
	client *client.Client
}

func (r *edgeConfigItemResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_config_item"
}

func (r *edgeConfigItemResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Schema returns the schema information for an edgeConfigToken resource.
func (r *edgeConfigItemResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides an Edge Config Item.

An Edge Config is a global data store that enables experimentation with feature flags, A/B testing, critical redirects, and more.

An Edge Config Item is a value within an Edge Config.
`,
		Attributes: map[string]schema.Attribute{
			"edge_config_id": schema.StringAttribute{
				Description:   "The ID of the Edge Config store.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Edge Config should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"key": schema.StringAttribute{
				Description:   "The name of the key you want to add to or update within your Edge Config.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"value": schema.StringAttribute{
				Description:   "The value you want to assign to the key.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

type EdgeConfigItem struct {
	EdgeConfigID types.String `tfsdk:"edge_config_id"`
	TeamID       types.String `tfsdk:"team_id"`
	Key          types.String `tfsdk:"key"`
	Value        types.String `tfsdk:"value"`
}

func responseToEdgeConfigItem(out client.EdgeConfigItem) EdgeConfigItem {
	return EdgeConfigItem{
		EdgeConfigID: types.StringValue(out.EdgeConfigID),
		TeamID:       types.StringValue(out.TeamID),
		Key:          types.StringValue(out.Key),
		Value:        types.StringValue(out.Value),
	}
}

// Create will create an edgeConfigToken within Vercel.
// This is called automatically by the provider when a new resource should be created.
func (r *edgeConfigItemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EdgeConfigItem
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateEdgeConfigItem(ctx, client.CreateEdgeConfigItemRequest{
		TeamID:       plan.TeamID.ValueString(),
		EdgeConfigID: plan.EdgeConfigID.ValueString(),
		Key:          plan.Key.ValueString(),
		Value:        plan.Value.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Edge Config Item",
			"Could not create Edge Config Item, unexpected error: "+err.Error(),
		)
		return
	}

	result := responseToEdgeConfigItem(out)
	tflog.Info(ctx, "created Edge Config Item", map[string]interface{}{
		"edge_config_id": plan.EdgeConfigID.ValueString(),
		"key":            result.Key.ValueString(),
		"value":          result.Value.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read edgeConfigToken information by requesting it from the Vercel API, and will update terraform
// with this information.
func (r *edgeConfigItemResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EdgeConfigItem
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetEdgeConfigItem(ctx, client.EdgeConfigItemRequest{
		EdgeConfigID: state.EdgeConfigID.ValueString(),
		TeamID:       state.TeamID.ValueString(),
		Key:          state.Key.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Edge Config Item",
			fmt.Sprintf("Could not get Edge Config Item %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.Key.ValueString(),
				err,
			),
		)
		return
	}

	result := responseToEdgeConfigItem(out)
	tflog.Info(ctx, "read edge config token", map[string]interface{}{
		"edge_config_id": state.EdgeConfigID.ValueString(),
		"team_id":        state.TeamID.ValueString(),
		"key":            state.Key.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update is the same as Create
func (r *edgeConfigItemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	panic(errors.New("Update is not supported, attributes require replacement"))
}

// Delete deletes an Edge Config Item.
func (r *edgeConfigItemResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EdgeConfigItem
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteEdgeConfigItem(ctx, client.EdgeConfigItemRequest{
		TeamID:       state.TeamID.ValueString(),
		EdgeConfigID: state.EdgeConfigID.ValueString(),
		Key:          state.Key.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Edge Config Item",
			fmt.Sprintf(
				"Could not delete Edge Config Item %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.EdgeConfigID.ValueString(),
				state.Key.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted edge config token", map[string]interface{}{
		"edge_config_id": state.EdgeConfigID.ValueString(),
		"team_id":        state.TeamID.ValueString(),
		"key":            state.Key.ValueString(),
	})
}

func (r *edgeConfigItemResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, edgeConfigId, id, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Edge Config Item",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/edge_config_id/key\" or \"edge_config_id/key\"", req.ID),
		)
	}

	out, err := r.client.GetEdgeConfigItem(ctx, client.EdgeConfigItemRequest{
		EdgeConfigID: edgeConfigId,
		TeamID:       teamID,
		Key:          id,
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Edge Config Item",
			fmt.Sprintf("Could not get Edge Config Item %s %s %s, unexpected error: %s",
				teamID,
				edgeConfigId,
				id,
				err,
			),
		)
		return
	}

	result := responseToEdgeConfigItem(out)
	tflog.Info(ctx, "import edge config schema", map[string]interface{}{
		"team_id":        result.TeamID.ValueString(),
		"edge_config_id": result.EdgeConfigID.ValueString(),
		"key":            result.Key.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
