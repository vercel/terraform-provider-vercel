package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &attackChallengeModeResource{}
	_ resource.ResourceWithConfigure   = &attackChallengeModeResource{}
	_ resource.ResourceWithImportState = &attackChallengeModeResource{}
)

func newAttackChallengeModeResource() resource.Resource {
	return &attackChallengeModeResource{}
}

type attackChallengeModeResource struct {
	client *client.Client
}

func (r *attackChallengeModeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_attack_challenge_mode"
}

func (r *attackChallengeModeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *attackChallengeModeResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides an Attack Challenge Mode resource.

Attack Challenge Mode prevent malicious traffic by showing a verification challenge for every visitor.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The resource identifier.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Description:   "The ID of the Project to toggle Attack Challenge Mode on.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Project exists under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"enabled": schema.BoolAttribute{
				Required:    true,
				Description: "Whether Attack Challenge Mode is enabled or not.",
			},
		},
	}
}

type AttackChallengeMode struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	TeamID    types.String `tfsdk:"team_id"`
	Enabled   types.Bool   `tfsdk:"enabled"`
}

func responseToAttackChallengeMode(out client.AttackChallengeMode) AttackChallengeMode {
	return AttackChallengeMode{
		ID:        types.StringValue(out.ProjectID),
		ProjectID: types.StringValue(out.ProjectID),
		TeamID:    toTeamID(out.TeamID),
		Enabled:   types.BoolValue(out.Enabled),
	}
}

func (r *attackChallengeModeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AttackChallengeMode
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.GetProject(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString())
	if client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Error creating Attack Challenge Mode",
			"Could not find project, please make sure both the project_id and team_id match the project and team you wish to deploy to.",
		)
		return
	}
	out, err := r.client.UpdateAttackChallengeMode(ctx, client.AttackChallengeMode{
		TeamID:    plan.TeamID.ValueString(),
		ProjectID: plan.ProjectID.ValueString(),
		Enabled:   plan.Enabled.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Attack Challenge Mode",
			"Could not create Attack Challenge Mode, unexpected error: "+err.Error(),
		)
		return
	}

	result := responseToAttackChallengeMode(out)
	tflog.Info(ctx, "created attack challenge mode", map[string]any{
		"team_id":    plan.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *attackChallengeModeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AttackChallengeMode
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetAttackChallengeMode(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Attack Challenge Mode",
			fmt.Sprintf("Could not get Attack Challenge Mode %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := responseToAttackChallengeMode(out)
	tflog.Info(ctx, "read attack challenge mode", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update does nothing.
func (r *attackChallengeModeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AttackChallengeMode
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateAttackChallengeMode(ctx, client.AttackChallengeMode{
		TeamID:    plan.TeamID.ValueString(),
		ProjectID: plan.ProjectID.ValueString(),
		Enabled:   plan.Enabled.ValueBool(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Attack Challenge Mode",
			fmt.Sprintf("Could not update Attack Challenge Mode %s %s, unexpected error: %s",
				plan.TeamID.ValueString(),
				plan.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := responseToAttackChallengeMode(out)
	tflog.Trace(ctx, "update attack challenge mode", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *attackChallengeModeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AttackChallengeMode
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Disable on deletion
	_, err := r.client.UpdateAttackChallengeMode(ctx, client.AttackChallengeMode{
		TeamID:    state.TeamID.ValueString(),
		ProjectID: state.ProjectID.ValueString(),
		Enabled:   false,
	})
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Attack Challenge Mode",
			fmt.Sprintf(
				"Could not delete Attack Challenge Mode %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted attack challenge mode", map[string]any{
		"team_id":    state.TeamID.ValueString(),
		"project_id": state.ProjectID.ValueString(),
	})
}

func (r *attackChallengeModeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Attack Challenge Mode",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id\" or \"project_id\"", req.ID),
		)
	}

	out, err := r.client.GetAttackChallengeMode(ctx, projectID, teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Attack Challenge Mode",
			fmt.Sprintf("Could not get Attack Challenge Mode %s %s, unexpected error: %s",
				teamID,
				projectID,
				err,
			),
		)
		return
	}

	result := responseToAttackChallengeMode(out)
	tflog.Info(ctx, "import attack challenge mode", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
