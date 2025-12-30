package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

var (
	_ resource.Resource                = &microfrontendGroupMembershipResource{}
	_ resource.ResourceWithConfigure   = &microfrontendGroupMembershipResource{}
	_ resource.ResourceWithImportState = &microfrontendGroupMembershipResource{}
)

func newMicrofrontendGroupMembershipResource() resource.Resource {
	return &microfrontendGroupMembershipResource{}
}

type microfrontendGroupMembershipResource struct {
	client *client.Client
}

func (r *microfrontendGroupMembershipResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_microfrontend_group_membership"
}

func (r *microfrontendGroupMembershipResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func getMicrofrontendGroupMembershipSchema(isDefaultApp bool) map[string]schema.Attribute {
	res := map[string]schema.Attribute{}

	res["id"] = schema.StringAttribute{
		Description: "The unique identifier for this resource. Format: team_id/microfrontend_group_id.",
		Computed:    true,
	}
	res["project_id"] = schema.StringAttribute{
		Description:   "The ID of the project.",
		Required:      true,
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	res["default_route"] = schema.StringAttribute{
		Description:   "The default route for the project. Used for the screenshot of deployments.",
		Optional:      true,
		Computed:      true,
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
	}

	if !isDefaultApp {
		res["microfrontend_group_id"] = schema.StringAttribute{
			Description:   "The ID of the microfrontend group.",
			Required:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		}
		res["team_id"] = schema.StringAttribute{
			Description:   "The team ID to add the microfrontend group to. Required when configuring a team resource if a default team has not been set in the provider.",
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
		}
		res["route_observability_to_this_project"] = schema.BoolAttribute{
			Description:   "Whether the project is route observability for this project. If dalse, the project will be route observability for all projects to the default project.",
			Optional:      true,
			Computed:      true,
			Default:       booldefault.StaticBool(true),
			PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseNonNullStateForUnknown()},
		}
	}

	return res
}

// Schema returns the schema information for a microfrontendGroupMembership resource.
func (r *microfrontendGroupMembershipResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Microfrontend Group Membership resource.

A Microfrontend Group Membership is a definition of a Vercel Project being a part of a Microfrontend Group.
`,
		Attributes: getMicrofrontendGroupMembershipSchema(false),
	}
}

type MicrofrontendGroupMembership struct {
	ID                              types.String `tfsdk:"id"`
	ProjectID                       types.String `tfsdk:"project_id"`
	MicrofrontendGroupID            types.String `tfsdk:"microfrontend_group_id"`
	TeamID                          types.String `tfsdk:"team_id"`
	DefaultRoute                    types.String `tfsdk:"default_route"`
	RouteObservabilityToThisProject types.Bool   `tfsdk:"route_observability_to_this_project"`
}

func convertResponseToMicrofrontendGroupMembership(membership client.MicrofrontendGroupMembership) MicrofrontendGroupMembership {
	return MicrofrontendGroupMembership{
		ID:                              types.StringValue(membership.TeamID + "/" + membership.MicrofrontendGroupID),
		ProjectID:                       types.StringValue(membership.ProjectID),
		MicrofrontendGroupID:            types.StringValue(membership.MicrofrontendGroupID),
		TeamID:                          types.StringValue(membership.TeamID),
		DefaultRoute:                    types.StringValue(membership.DefaultRoute),
		RouteObservabilityToThisProject: types.BoolValue(membership.RouteObservabilityToThisProject),
	}
}

func (r *microfrontendGroupMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MicrofrontendGroupMembership
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError(
			"Error getting microfrontend group membership plan",
			"Error getting microfrontend group membership plan",
		)
		return
	}

	tflog.Info(ctx, "creating microfrontend group membership", map[string]any{
		"project_id": plan.ProjectID.ValueString(),
		"group_id":   plan.MicrofrontendGroupID.ValueString(),
		"plan":       plan,
	})

	out, err := r.client.AddOrUpdateMicrofrontendGroupMembership(ctx, client.MicrofrontendGroupMembership{
		ProjectID:                       plan.ProjectID.ValueString(),
		MicrofrontendGroupID:            plan.MicrofrontendGroupID.ValueString(),
		DefaultRoute:                    plan.DefaultRoute.ValueString(),
		RouteObservabilityToThisProject: plan.RouteObservabilityToThisProject.ValueBool(),
		TeamID:                          plan.TeamID.ValueString(),
		IsDefaultApp:                    false,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating microfrontend group membership",
			"Could not create microfrontend group, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToMicrofrontendGroupMembership(out)
	tflog.Info(ctx, "created microfrontend group membership", map[string]any{
		"project_id": result.ProjectID.ValueString(),
		"group_id":   result.MicrofrontendGroupID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *microfrontendGroupMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MicrofrontendGroupMembership
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ProjectID.ValueString() == "" || state.MicrofrontendGroupID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	out, err := r.client.GetMicrofrontendGroupMembership(ctx,
		state.TeamID.ValueString(),
		state.MicrofrontendGroupID.ValueString(),
		state.ProjectID.ValueString(),
	)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading microfrontend group membership",
			fmt.Sprintf("Could not get microfrontend group membership %s %s, unexpected error: %s",
				state.ProjectID.ValueString(),
				state.MicrofrontendGroupID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToMicrofrontendGroupMembership(out)
	tflog.Info(ctx, "read microfrontend group membership", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"group_id":   result.MicrofrontendGroupID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *microfrontendGroupMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MicrofrontendGroupMembership
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError(
			"Error getting microfrontend group plan",
			"Error getting microfrontend group plan",
		)
		return
	}

	var state MicrofrontendGroupMembership
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.AddOrUpdateMicrofrontendGroupMembership(ctx, client.MicrofrontendGroupMembership{
		ProjectID:                       plan.ProjectID.ValueString(),
		MicrofrontendGroupID:            plan.MicrofrontendGroupID.ValueString(),
		DefaultRoute:                    plan.DefaultRoute.ValueString(),
		RouteObservabilityToThisProject: plan.RouteObservabilityToThisProject.ValueBool(),
		TeamID:                          plan.TeamID.ValueString(),
		IsDefaultApp:                    false,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating microfrontend group membership",
			fmt.Sprintf(
				"Could not update microfrontend group membership %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.MicrofrontendGroupID.ValueString(),
				state.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "updated microfrontend group membership", map[string]any{
		"team_id":                out.TeamID,
		"microfrontend_group_id": out.MicrofrontendGroupID,
		"project_id":             out.ProjectID,
	})

	result := convertResponseToMicrofrontendGroupMembership(out)

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *microfrontendGroupMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MicrofrontendGroupMembership
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "deleting microfrontend group membership", map[string]any{
		"project_id": state.ProjectID.ValueString(),
		"group_id":   state.MicrofrontendGroupID.ValueString(),
		"team_id":    state.TeamID.ValueString(),
	})

	_, err := r.client.RemoveMicrofrontendGroupMembership(ctx, client.MicrofrontendGroupMembership{
		MicrofrontendGroupID: state.MicrofrontendGroupID.ValueString(),
		ProjectID:            state.ProjectID.ValueString(),
		TeamID:               state.TeamID.ValueString(),
		IsDefaultApp:         false,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting microfrontend group membership",
			fmt.Sprintf(
				"Could not delete microfrontend group membership %s %s, unexpected error: %s",
				state.MicrofrontendGroupID.ValueString(),
				state.ProjectID.ValueString(),
				err,
			),
		)
		return
	}
	tflog.Info(ctx, "deleted microfrontend group membership", map[string]any{
		"group_id":   state.MicrofrontendGroupID.ValueString(),
		"project_id": state.ProjectID.ValueString(),
	})
}

func (r *microfrontendGroupMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, microfrontendID, projectID, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Microfrontend Group Membership",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/microfrontend_id/project_id\" or \"microfrontend_id/project_id\"", req.ID),
		)
	}
	out, err := r.client.GetMicrofrontendGroupMembership(ctx, teamID, microfrontendID, projectID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing microfrontend group membership",
			fmt.Sprintf("Could not import microfrontend group membership %s %s %s, unexpected error: %s",
				teamID,
				microfrontendID,
				projectID,
				err,
			),
		)
		return
	}

	result := convertResponseToMicrofrontendGroupMembership(out)
	tflog.Info(ctx, "import microfrontend group", map[string]any{
		"project_id": result.ProjectID.ValueString(),
		"group_id":   result.MicrofrontendGroupID.ValueString(),
		"team_id":    result.TeamID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
