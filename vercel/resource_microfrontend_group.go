package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

var (
	_ resource.Resource                = &microfrontendGroupResource{}
	_ resource.ResourceWithConfigure   = &microfrontendGroupResource{}
	_ resource.ResourceWithImportState = &microfrontendGroupResource{}
)

func newMicrofrontendGroupResource() resource.Resource {
	return &microfrontendGroupResource{}
}

type microfrontendGroupResource struct {
	client *client.Client
}

func (r *microfrontendGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_microfrontend_group"
}

func (r *microfrontendGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Schema returns the schema information for a microfrontendGroup resource.
func (r *microfrontendGroupResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Microfrontend Group resource.

A Microfrontend Group is a definition of a microfrontend belonging to a Vercel Team.
`,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "A human readable name for the microfrontends group.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description:   "A unique identifier for the group of microfrontends. Example: mfe_12HKQaOmR5t5Uy6vdcQsNIiZgHGB",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"slug": schema.StringAttribute{
				Description: "A slugified version of the name.",
				Computed:    true,
			},
			"team_id": schema.StringAttribute{
				Description:   "The team ID to add the microfrontend group to. Required when configuring a team resource if a default team has not been set in the provider.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"default_app": schema.SingleNestedAttribute{
				Description: "The default app for the project. Used as the entry point for the microfrontend.",
				Required:    true,
				Attributes:  getMicrofrontendGroupMembershipSchema(true),
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplaceIf(func(ctx context.Context, req planmodifier.ObjectRequest, resp *objectplanmodifier.RequiresReplaceIfFuncResponse) {
						oldDefaultApp, okOld := req.ConfigValue.ToObjectValue(ctx)
						newDefaultApp, okNew := req.PlanValue.ToObjectValue(ctx)
						if okOld.HasError() || okNew.HasError() {
							return
						}
						oldValue := oldDefaultApp.Attributes()["project_id"]
						newValue := newDefaultApp.Attributes()["project_id"]

						if oldValue != newValue {
							resp.RequiresReplace = true
						}
					}, "The default app for the group has changed.", "The default app for the group has changed."),
				},
			},
		},
	}
}

type MicrofrontendGroupDefaultApp struct {
	ProjectID    types.String `tfsdk:"project_id"`
	DefaultRoute types.String `tfsdk:"default_route"`
}

type MicrofrontendGroup struct {
	TeamID     types.String `tfsdk:"team_id"`
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Slug       types.String `tfsdk:"slug"`
	DefaultApp types.Object `tfsdk:"default_app"`
}

var microfrontendDefaultAppAttrType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"project_id":    types.StringType,
		"default_route": types.StringType,
	},
}

func convertResponseToMicrofrontendGroup(group client.MicrofrontendGroup) MicrofrontendGroup {
	return MicrofrontendGroup{
		ID:     types.StringValue(group.ID),
		Name:   types.StringValue(group.Name),
		Slug:   types.StringValue(group.Slug),
		TeamID: types.StringValue(group.TeamID),
		DefaultApp: types.ObjectValueMust(microfrontendDefaultAppAttrType.AttrTypes, map[string]attr.Value{
			"project_id":    types.StringValue(group.DefaultApp.ProjectID),
			"default_route": types.StringValue(group.DefaultApp.DefaultRoute),
		}),
	}
}

func (r *microfrontendGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MicrofrontendGroup
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError(
			"Error getting microfrontend group plan",
			"Error getting microfrontend group plan",
		)
		return
	}

	tflog.Info(ctx, "creating microfrontend group", map[string]any{
		"team_id": plan.TeamID.ValueString(),
		"name":    plan.Name.ValueString(),
	})

	out, err := r.client.CreateMicrofrontendGroup(ctx, plan.TeamID.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating microfrontend group",
			"Could not create microfrontend group, unexpected error: "+err.Error(),
		)
		return
	}

	var da MicrofrontendGroupDefaultApp
	_ = plan.DefaultApp.As(ctx, &da, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
	tflog.Info(ctx, "creating default group membership", map[string]any{
		"team_id":     plan.TeamID.ValueString(),
		"name":        plan.Name.ValueString(),
		"default_app": da.ProjectID.ValueString(),
	})

	default_app, err := r.client.AddOrUpdateMicrofrontendGroupMembership(ctx, client.MicrofrontendGroupMembership{
		ProjectID:            da.ProjectID.ValueString(),
		MicrofrontendGroupID: out.ID,
		TeamID:               plan.TeamID.ValueString(),
		DefaultRoute:         da.DefaultRoute.ValueString(),
		IsDefaultApp:         true,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating microfrontend default app group membership",
			"Could not create microfrontend default app group membership, unexpected error: "+err.Error(),
		)
		return
	}

	group := client.MicrofrontendGroup{
		ID:     out.ID,
		Name:   out.Name,
		Slug:   out.Slug,
		TeamID: out.TeamID,
		DefaultApp: client.MicrofrontendGroupMembership{
			ProjectID:                       default_app.ProjectID,
			TeamID:                          default_app.TeamID,
			DefaultRoute:                    default_app.DefaultRoute,
			RouteObservabilityToThisProject: default_app.RouteObservabilityToThisProject,
			MicrofrontendGroupID:            out.ID,
			IsDefaultApp:                    default_app.IsDefaultApp,
		},
		Projects: out.Projects,
	}

	result := convertResponseToMicrofrontendGroup(group)
	var resDa MicrofrontendGroupDefaultApp
	_ = result.DefaultApp.As(ctx, &resDa, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
	tflog.Info(ctx, "created microfrontend group", map[string]any{
		"team_id":     result.TeamID.ValueString(),
		"group_id":    result.ID.ValueString(),
		"slug":        result.Slug.ValueString(),
		"name":        result.Name.ValueString(),
		"default_app": resDa.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *microfrontendGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MicrofrontendGroup
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetMicrofrontendGroup(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading microfrontend group",
			fmt.Sprintf("Could not get microfrontend group %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToMicrofrontendGroup(out)
	var resDa MicrofrontendGroupDefaultApp
	_ = result.DefaultApp.As(ctx, &resDa, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
	tflog.Info(ctx, "read microfrontend group", map[string]any{
		"defaultApp": resDa.ProjectID.ValueString(),
		"team_id":    result.TeamID.ValueString(),
		"group_id":   result.ID.ValueString(),
		"slug":       result.Slug.ValueString(),
		"name":       result.Name.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *microfrontendGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MicrofrontendGroup
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError(
			"Error getting microfrontend group plan",
			"Error getting microfrontend group plan",
		)
		return
	}

	var state MicrofrontendGroup
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateMicrofrontendGroup(ctx, client.MicrofrontendGroup{
		ID:     state.ID.ValueString(),
		Name:   plan.Name.ValueString(),
		TeamID: state.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating microfrontend group",
			fmt.Sprintf(
				"Could not update microfrontend group %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "updated microfrontend group", map[string]any{
		"team_id":  out.TeamID,
		"group_id": out.ID,
		"name":     out.Name,
		"slug":     out.Slug,
	})

	result := convertResponseToMicrofrontendGroup(out)

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *microfrontendGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MicrofrontendGroup
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stDa MicrofrontendGroupDefaultApp
	_ = state.DefaultApp.As(ctx, &stDa, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
	tflog.Info(ctx, "deleting microfrontend default app group membership", map[string]any{
		"group_id":   state.ID.ValueString(),
		"project_id": stDa.ProjectID.ValueString(),
		"team_id":    state.TeamID.ValueString(),
	})

	_, err := r.client.RemoveMicrofrontendGroupMembership(ctx, client.MicrofrontendGroupMembership{
		MicrofrontendGroupID: state.ID.ValueString(),
		TeamID:               state.TeamID.ValueString(),
		ProjectID:            stDa.ProjectID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting microfrontend default app group membership",
			fmt.Sprintf(
				"Could not delete microfrontend default app group membership %s %s, unexpected error: %s",
				state.ID.ValueString(),
				stDa.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleting microfrontend group", map[string]any{
		"group_id": state.ID.ValueString(),
	})

	_, err = r.client.DeleteMicrofrontendGroup(ctx, client.MicrofrontendGroup{
		ID:     state.ID.ValueString(),
		TeamID: state.TeamID.ValueString(),
		Slug:   state.Slug.ValueString(),
		Name:   state.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting microfrontend group",
			fmt.Sprintf(
				"Could not delete microfrontend group %s, unexpected error: %s",
				state.ID.ValueString(),
				err,
			),
		)
		return
	}
	tflog.Info(ctx, "deleted microfrontendGroup", map[string]any{
		"group_id": state.ID.ValueString(),
	})
}

func (r *microfrontendGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, microfrontendID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Microfrontend Group",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/microfrontend_id\" or \"microfrontend_id\"", req.ID),
		)
	}
	out, err := r.client.GetMicrofrontendGroup(ctx, microfrontendID, teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing microfrontend group",
			fmt.Sprintf("Could not import microfrontend group %s %s, unexpected error: %s",
				teamID,
				microfrontendID,
				err,
			),
		)
		return
	}

	result := convertResponseToMicrofrontendGroup(out)
	var resDa2 MicrofrontendGroupDefaultApp
	_ = result.DefaultApp.As(ctx, &resDa2, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
	tflog.Info(ctx, "import microfrontend group", map[string]any{
		"defaultApp": resDa2.ProjectID.ValueString(),
		"team_id":    result.TeamID.ValueString(),
		"group_id":   result.ID.ValueString(),
		"slug":       result.Slug.ValueString(),
		"name":       result.Name.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
