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
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &aliasResource{}
	_ resource.ResourceWithConfigure = &aliasResource{}
)

func newAliasResource() resource.Resource {
	return &aliasResource{}
}

type aliasResource struct {
	client *client.Client
}

func (r *aliasResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alias"
}

func (r *aliasResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Schema returns the schema information for an alias resource.
func (r *aliasResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides an Alias resource.

An Alias allows a ` + "`vercel_deployment` to be accessed through a different URL.",
		Attributes: map[string]schema.Attribute{
			"alias": schema.StringAttribute{
				Description: "The Alias we want to assign to the deployment defined in the URL.",
				Required:    true,
			},
			"deployment_id": schema.StringAttribute{
				Description: "The id of the Deployment the Alias should be associated with.",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the Alias and Deployment exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Alias represents the terraform state for an alias resource.
type Alias struct {
	Alias        types.String `tfsdk:"alias"`
	ID           types.String `tfsdk:"id"`
	DeploymentID types.String `tfsdk:"deployment_id"`
	TeamID       types.String `tfsdk:"team_id"`
}

// convertResponseToAlias is used to populate terraform state based on an API response.
// Where possible, values from the API response are used to populate state. If not possible,
// values from plan are used.
func convertResponseToAlias(response client.AliasResponse, plan Alias) Alias {
	return Alias{
		Alias:        plan.Alias,
		ID:           types.StringValue(response.UID),
		DeploymentID: types.StringValue(response.DeploymentID),
		TeamID:       toTeamID(response.TeamID),
	}
}

// Create will create an alias within Vercel.
// This is called automatically by the provider when a new resource should be created.
func (r *aliasResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan Alias
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use the update function to create and update aliases, as the API is an upsert
	out, err := r.client.UpsertAlias(ctx, client.UpsertAliasRequest{
		Alias:        plan.Alias.ValueString(),
		DeploymentID: plan.DeploymentID.ValueString(),
		TeamID:       plan.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating alias",
			"Could not create alias, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToAlias(out, plan)
	tflog.Info(ctx, "created alias", map[string]any{
		"team_id":       plan.TeamID.ValueString(),
		"deployment_id": plan.DeploymentID.ValueString(),
		"alias_id":      result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read alias information by requesting it from the Vercel API, and will update terraform
// with this information.
func (r *aliasResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state Alias
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetAlias(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading alias",
			fmt.Sprintf("Could not get alias %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToAlias(out, state)
	tflog.Info(ctx, "read alias", map[string]any{
		"team_id":  result.TeamID.ValueString(),
		"alias_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the Alias state.
// The Vercel API for creating an alias is an upsert. We can simply call the create method again to update.
func (r *aliasResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan Alias
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use the update function to create and update aliases, as the API is an upsert
	out, err := r.client.UpsertAlias(ctx, client.UpsertAliasRequest{
		Alias:        plan.Alias.ValueString(),
		DeploymentID: plan.DeploymentID.ValueString(),
		TeamID:       plan.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating alias",
			fmt.Sprintf("Could not update alias %s, unexpected error: %s", plan.Alias.ValueString(), err.Error()),
		)
		return
	}

	result := convertResponseToAlias(out, plan)
	tflog.Info(ctx, "updated alias", map[string]any{
		"team_id":       plan.TeamID.ValueString(),
		"deployment_id": plan.DeploymentID.ValueString(),
		"alias_id":      result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes an Alias.
func (r *aliasResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state Alias
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteAlias(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting alias",
			fmt.Sprintf(
				"Could not delete alias %s, unexpected error: %s",
				state.Alias.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted alias", map[string]any{
		"team_id":  state.TeamID.ValueString(),
		"alias_id": state.ID.ValueString(),
	})
}
