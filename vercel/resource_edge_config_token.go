package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &edgeConfigTokenResource{}
	_ resource.ResourceWithConfigure   = &edgeConfigTokenResource{}
	_ resource.ResourceWithImportState = &edgeConfigTokenResource{}
)

func newEdgeConfigTokenResource() resource.Resource {
	return &edgeConfigTokenResource{}
}

type edgeConfigTokenResource struct {
	client *client.Client
}

func (r *edgeConfigTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_config_token"
}

func (r *edgeConfigTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *edgeConfigTokenResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides an Edge Config Token resource.

An Edge Config is a global data store that enables experimentation with feature flags, A/B testing, critical redirects, and more.

An Edge Config token is used to authenticate against an Edge Config's endpoint.
`,
		Attributes: map[string]schema.Attribute{
			"label": schema.StringAttribute{
				Description:   "The label of the Edge Config Token.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 52),
				},
			},
			"edge_config_id": schema.StringAttribute{
				Description:   "The ID of the Edge Config store.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Edge Config should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"id": schema.StringAttribute{
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
				Computed:      true,
			},
			"token": schema.StringAttribute{
				Description: "A read access token used for authenticating against the Edge Config's endpoint for high volume, low-latency requests.",
				Computed:    true,
			},
			"connection_string": schema.StringAttribute{
				Description: "A connection string is a URL that connects a project to an Edge Config. The variable can be called anything, but our Edge Config client SDK will search for process.env.EDGE_CONFIG by default.",
				Computed:    true,
			},
		},
	}
}

type EdgeConfigToken struct {
	Label            types.String `tfsdk:"label"`
	Token            types.String `tfsdk:"token"`
	ID               types.String `tfsdk:"id"`
	TeamID           types.String `tfsdk:"team_id"`
	EdgeConfigID     types.String `tfsdk:"edge_config_id"`
	ConnectionString types.String `tfsdk:"connection_string"`
}

func responseToEdgeConfigToken(out client.EdgeConfigToken) EdgeConfigToken {
	return EdgeConfigToken{
		TeamID:           toTeamID(out.TeamID),
		Token:            types.StringValue(out.Token),
		Label:            types.StringValue(out.Label),
		ID:               types.StringValue(out.ID),
		EdgeConfigID:     types.StringValue(out.EdgeConfigID),
		ConnectionString: types.StringValue(out.ConnectionString()),
	}
}

// Create will create an edgeConfigToken within Vercel.
// This is called automatically by the provider when a new resource should be created.
func (r *edgeConfigTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EdgeConfigToken
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateEdgeConfigToken(ctx, client.CreateEdgeConfigTokenRequest{
		Label:        plan.Label.ValueString(),
		TeamID:       plan.TeamID.ValueString(),
		EdgeConfigID: plan.EdgeConfigID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Edge Config Token",
			"Could not create Edge Config Token, unexpected error: "+err.Error(),
		)
		return
	}

	result := responseToEdgeConfigToken(out)
	tflog.Info(ctx, "created Edge Config Token", map[string]any{
		"team_id":        plan.TeamID.ValueString(),
		"edge_config_id": result.EdgeConfigID.ValueString(),
		"token_id":       result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read edgeConfigToken information by requesting it from the Vercel API, and will update terraform
// with this information.
func (r *edgeConfigTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EdgeConfigToken
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetEdgeConfigToken(ctx, client.EdgeConfigTokenRequest{
		Token:        state.Token.ValueString(),
		TeamID:       state.TeamID.ValueString(),
		EdgeConfigID: state.EdgeConfigID.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Edge Config Token",
			fmt.Sprintf("Could not get Edge Config Token %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := responseToEdgeConfigToken(out)
	tflog.Info(ctx, "read edge config token", map[string]any{
		"team_id":        result.TeamID.ValueString(),
		"edge_config_id": result.EdgeConfigID.ValueString(),
		"token_id":       result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update does nothing.
func (r *edgeConfigTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Updating an Edge Config Token is not supported",
		"Updating an Edge Config Token is not supported",
	)
}

// Delete deletes an Edge Config.
func (r *edgeConfigTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EdgeConfigToken
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteEdgeConfigToken(ctx, client.EdgeConfigTokenRequest{
		Token:        state.Token.ValueString(),
		TeamID:       state.TeamID.ValueString(),
		EdgeConfigID: state.EdgeConfigID.ValueString(),
	})
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Edge Config Token",
			fmt.Sprintf(
				"Could not delete Edge Config Token %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.EdgeConfigID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted edge config token", map[string]any{
		"team_id":        state.TeamID.ValueString(),
		"edge_config_id": state.EdgeConfigID.ValueString(),
		"token_id":       state.ID.ValueString(),
	})
}

func (r *edgeConfigTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, edgeConfigID, token, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing edge config token",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/edge_config_id/token\" or \"edge_config_id/token\"", req.ID),
		)
	}

	out, err := r.client.GetEdgeConfigToken(ctx, client.EdgeConfigTokenRequest{
		Token:        token,
		EdgeConfigID: edgeConfigID,
		TeamID:       teamID,
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing Edge Config Token",
			fmt.Sprintf("Could not get Edge Config Token %s %s %s, unexpected error: %s",
				teamID,
				edgeConfigID,
				token,
				err,
			),
		)
		return
	}

	result := responseToEdgeConfigToken(out)
	tflog.Info(ctx, "import edge config token", map[string]any{
		"team_id":        result.TeamID.ValueString(),
		"edge_config_id": result.EdgeConfigID.ValueString(),
		"token_id":       result.ID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
