package vercel

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

var (
	_ resource.Resource              = &sharedEnvironmentVariableResource{}
	_ resource.ResourceWithConfigure = &sharedEnvironmentVariableResource{}
)

func newSharedEnvironmentVariableResource() resource.Resource {
	return &sharedEnvironmentVariableResource{}
}

type sharedEnvironmentVariableResource struct {
	client *client.Client
}

func (r *sharedEnvironmentVariableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shared_environment_variable"
}

func (r *sharedEnvironmentVariableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Schema returns the schema information for a shared environment variable resource.
func (r *sharedEnvironmentVariableResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Shared Environment Variable resource.

A Shared Environment Variable resource defines an Environment Variable that can be shared between multiple Vercel Projects.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/concepts/projects/environment-variables/shared-environment-variables).
`,
		Attributes: map[string]schema.Attribute{
			"target": schema.SetAttribute{
				Required:    true,
				Description: "The environments that the Environment Variable should be present on. Valid targets are either `production`, `preview`, or `development`.",
				ElementType: types.StringType,
				Validators: []validator.Set{
					stringSetItemsIn("production", "preview", "development"),
					stringSetMinCount(1),
				},
			},
			"key": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "The name of the Environment Variable.",
			},
			"value": schema.StringAttribute{
				Required:    true,
				Description: "The value of the Environment Variable.",
				Sensitive:   true,
			},
			"project_ids": schema.SetAttribute{
				Required:    true,
				Description: "The ID of the Vercel project.",
				ElementType: types.StringType,
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the Vercel team. Shared environment variables require a team.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"id": schema.StringAttribute{
				Description:   "The ID of the Environment Variable.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
				Computed:      true,
			},
			"sensitive": schema.BoolAttribute{
				Description:   "Whether the Environment Variable is sensitive or not.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
		},
	}
}

// Create will create a new shared environment variable.
// This is called automatically by the provider when a new resource should be created.
func (r *sharedEnvironmentVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SharedEnvironmentVariable
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request, ok := plan.toCreateSharedEnvironmentVariableRequest(ctx, resp.Diagnostics)
	if !ok {
		return
	}
	response, err := r.client.CreateSharedEnvironmentVariable(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating shared environment variable",
			"Could not create shared environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToSharedEnvironmentVariable(response, plan.Value)

	tflog.Info(ctx, "created shared environment variable", map[string]interface{}{
		"id":      result.ID.ValueString(),
		"team_id": result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read an shared environment variable by requesting it from the Vercel API, and will update terraform
// with this information.
func (r *sharedEnvironmentVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SharedEnvironmentVariable
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetSharedEnvironmentVariable(ctx, state.TeamID.ValueString(), state.ID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading shared environment variable",
			fmt.Sprintf("Could not get shared environment variable %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToSharedEnvironmentVariable(out, state.Value)
	tflog.Info(ctx, "read shared environment variable", map[string]interface{}{
		"id":      result.ID.ValueString(),
		"team_id": result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the shared environment variable of a Vercel project state.
func (r *sharedEnvironmentVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SharedEnvironmentVariable
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	request, ok := plan.toUpdateSharedEnvironmentVariableRequest(ctx, resp.Diagnostics)
	if !ok {
		return
	}
	response, err := r.client.UpdateSharedEnvironmentVariable(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating shared environment variable",
			"Could not update shared environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToSharedEnvironmentVariable(response, plan.Value)

	tflog.Info(ctx, "updated shared environment variable", map[string]interface{}{
		"id":      result.ID.ValueString(),
		"team_id": result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes a Vercel shared environment variable.
func (r *sharedEnvironmentVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SharedEnvironmentVariable
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSharedEnvironmentVariable(ctx, state.TeamID.ValueString(), state.ID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting shared environment variable",
			fmt.Sprintf(
				"Could not delete shared environment variable %s, unexpected error: %s",
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted shared environment variable", map[string]interface{}{
		"id":      state.ID.ValueString(),
		"team_id": state.TeamID.ValueString(),
	})
}

// splitID is a helper function for splitting an import ID into the corresponding parts.
// It also validates whether the ID is in a correct format.
func splitSharedEnvironmentVariableID(id string) (teamID, envID string, ok bool) {
	attributes := strings.Split(id, "/")
	if len(attributes) == 2 {
		return attributes[0], attributes[1], true
	}

	return "", "", false
}

// ImportState takes an identifier and reads all the shared environment variable information from the Vercel API.
// The results are then stored in terraform state.
func (r *sharedEnvironmentVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, envID, ok := splitSharedEnvironmentVariableID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing shared environment variable",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/env_id\"", req.ID),
		)
	}

	out, err := r.client.GetSharedEnvironmentVariable(ctx, teamID, envID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading shared environment variable",
			fmt.Sprintf("Could not get shared environment variable %s %s, unexpected error: %s",
				teamID,
				envID,
				err,
			),
		)
		return
	}

	result := convertResponseToSharedEnvironmentVariable(out, types.StringNull())
	tflog.Info(ctx, "imported shared environment variable", map[string]interface{}{
		"team_id": result.TeamID.ValueString(),
		"env_id":  result.ID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
