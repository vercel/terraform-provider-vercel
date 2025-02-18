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
Projects are added to a Microfrontend Group.
`,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "A human readable name for the microfrontends group.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "A unique identifier for the group of microfrontends. Example: mfe_12HKQaOmR5t5Uy6vdcQsNIiZgHGB",
				Computed:    true,
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
		},
	}
}

// MicrofrontendGroup represents the terraform state for a microfrontendGroup resource.
type MicrofrontendGroup struct {
	TeamID types.String `tfsdk:"team_id"`
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Slug   types.String `tfsdk:"slug"`
}

// convertResponseToMicrofrontendGroup is used to populate terraform state based on an API response.
// Where possible, values from the API response are used to populate state. If not possible,
// values from the existing microfrontendGroup state are used.
func convertResponseToMicrofrontendGroup(response client.MicrofrontendGroupResponse) MicrofrontendGroup {
	return MicrofrontendGroup{
		ID:     types.StringValue(response.ID),
		Name:   types.StringValue(response.Name),
		Slug:   types.StringValue(response.Slug),
		TeamID: types.StringValue(response.TeamID),
	}
}

// Create will create a microfrontendGroup within Vercel. This is done by first attempting to trigger a microfrontendGroup, seeing what
// files are required, uploading those files, and then attempting to create a microfrontendGroup again.
// This is called automatically by the provider when a new resource should be created.
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

	cdr := client.CreateMicrofrontendGroupRequest{
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

	result := convertResponseToMicrofrontendGroup(out)
	tflog.Info(ctx, "created microfrontendGroup", map[string]interface{}{
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

// Read will read a file from the filesytem and provide terraform with information about it.
// It is called by the provider whenever data source values should be read to update state.
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
			"Error reading microfrontendGroup",
			fmt.Sprintf("Could not get microfrontendGroup %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToMicrofrontendGroup(out)
	tflog.Info(ctx, "read microfrontendGroup", map[string]interface{}{
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

// Update updates the microfrontendGroup state.
func (r *microfrontendGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	var state MicrofrontendGroup
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateMicrofrontendGroup(ctx, client.UpdateMicrofrontendGroupRequest{
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

// Delete deletes a MicrofrontendGroup.
func (r *microfrontendGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MicrofrontendGroup
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteMicrofrontendGroup(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting microfrontendGroup",
			fmt.Sprintf(
				"Could not delete microfrontendGroup %s, unexpected error: %s",
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
