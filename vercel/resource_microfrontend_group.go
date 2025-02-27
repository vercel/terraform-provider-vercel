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
	"github.com/vercel/terraform-provider-vercel/v2/client"
)

var (
	_ resource.Resource              = &microfrontendGroupResource{}
	_ resource.ResourceWithConfigure = &microfrontendGroupResource{}
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
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"slug": schema.StringAttribute{
				Description: "A slugified version of the name.",
				Computed:    true,
			},
			"team_id": schema.StringAttribute{
				Description:   "The team ID to add the microfrontend group to. Required when configuring a team resource if a default team has not been set in the provider.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"default_app": schema.StringAttribute{
				Description:   "The default app for the project. Used as the entry point for the microfrontend.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

type MicrofrontendGroup struct {
	TeamID     types.String `tfsdk:"team_id"`
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Slug       types.String `tfsdk:"slug"`
	DefaultApp types.String `tfsdk:"default_app"`
}

func convertResponseToMicrofrontendGroup(group client.MicrofrontendGroup) MicrofrontendGroup {
	return MicrofrontendGroup{
		ID:         types.StringValue(group.ID),
		Name:       types.StringValue(group.Name),
		Slug:       types.StringValue(group.Slug),
		TeamID:     types.StringValue(group.TeamID),
		DefaultApp: types.StringValue(group.DefaultApp),
	}
}

func (r *microfrontendGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MicrofrontendGroup
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError(
			"Error getting microfrontendGroup plan",
			"Error getting microfrontendGroup plan",
		)
		return
	}

	tflog.Info(ctx, "creating microfrontend group", map[string]interface{}{
		"team_id": plan.TeamID.ValueString(),
		"name":    plan.Name.ValueString(),
	})

	cdr := client.MicrofrontendGroup{
		Name:   plan.Name.ValueString(),
		TeamID: plan.TeamID.ValueString(),
	}

	out, err := r.client.CreateMicrofrontendGroup(ctx, cdr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating microfrontend group",
			"Could not create microfrontend group, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "creating default group membership", map[string]interface{}{
		"team_id":     plan.TeamID.ValueString(),
		"name":        plan.Name.ValueString(),
		"default_app": plan.DefaultApp.ValueString(),
	})

	group := client.MicrofrontendGroup{
		ID:         out.ID,
		Name:       out.Name,
		Slug:       out.Slug,
		TeamID:     out.TeamID,
		DefaultApp: plan.DefaultApp.ValueString(),
		Projects:   out.Projects,
	}

	_, err = r.client.AddOrUpdateMicrofrontendGroupMembership(ctx, client.MicrofrontendGroupMembership{
		ProjectID:            plan.DefaultApp.ValueString(),
		MicrofrontendGroupID: out.ID,
		TeamID:               plan.TeamID.ValueString(),
		IsDefaultApp:         true,
	}, group)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating microfrontend default app group membership",
			"Could not create microfrontend default app group membership, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToMicrofrontendGroup(group)
	tflog.Info(ctx, "created microfrontend group", map[string]interface{}{
		"team_id":     result.TeamID.ValueString(),
		"group_id":    result.ID.ValueString(),
		"slug":        result.Slug.ValueString(),
		"name":        result.Name.ValueString(),
		"default_app": result.DefaultApp.ValueString(),
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
	tflog.Info(ctx, "read microfrontend group", map[string]interface{}{
		"team_id":  result.TeamID.ValueString(),
		"group_id": result.ID.ValueString(),
		"slug":     result.Slug.ValueString(),
		"name":     result.Name.ValueString(),
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

	tflog.Info(ctx, "updated microfrontend group", map[string]interface{}{
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

	if state.DefaultApp.ValueString() != "" {
		tflog.Info(ctx, "deleting microfrontend default app group membership", map[string]interface{}{
			"group_id":   state.ID.ValueString(),
			"project_id": state.DefaultApp.ValueString(),
			"team_id":    state.TeamID.ValueString(),
		})

		_, err := r.client.RemoveMicrofrontendGroupMembership(ctx, client.MicrofrontendGroupMembership{
			MicrofrontendGroupID: state.ID.ValueString(),
			TeamID:               state.TeamID.ValueString(),
			ProjectID:            state.DefaultApp.ValueString(),
		}, true)

		if err != nil {
			resp.Diagnostics.AddError(
				"Error deleting microfrontend default app group membership",
				fmt.Sprintf(
					"Could not delete microfrontend default app group membership %s %s, unexpected error: %s",
					state.ID.ValueString(),
					state.DefaultApp.ValueString(),
					err,
				),
			)
			return
		}
	}

	tflog.Info(ctx, "deleting microfrontend group", map[string]interface{}{
		"group_id": state.ID.ValueString(),
	})

	_, err := r.client.DeleteMicrofrontendGroup(ctx, client.MicrofrontendGroup{
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
