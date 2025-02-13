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
	_ resource.Resource              = &integrationProjectAccessResource{}
	_ resource.ResourceWithConfigure = &integrationProjectAccessResource{}
)

func newIntegrationProjectAccessResource() resource.Resource {
	return &integrationProjectAccessResource{}
}

type integrationProjectAccessResource struct {
	client *client.Client
}

func (r *integrationProjectAccessResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_project_access"
}

func (r *integrationProjectAccessResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *integrationProjectAccessResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides Project access to an existing Integration. This requires the integration already exists and is already configured for Specific Project access.
`,
		Attributes: map[string]schema.Attribute{
			"integration_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "The ID of the integration.",
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the Vercel project.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the Vercel team.Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

type IntegrationProjectAccess struct {
	TeamID        types.String `tfsdk:"team_id"`
	ProjectID     types.String `tfsdk:"project_id"`
	IntegrationID types.String `tfsdk:"integration_id"`
}

func (r *integrationProjectAccessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IntegrationProjectAccess
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID := plan.TeamID
	if teamID.ValueString() == "" {
		t, err := r.client.Team(ctx, "")
		if err != nil {
			resp.Diagnostics.AddError(
				"Error getting team from provider",
				"Could not read team from provider, unexpected error: "+err.Error(),
			)
			return
		}
		teamID = types.StringValue(t.ID)
	}

	_, err := r.client.GrantIntegrationProjectAccess(ctx, plan.IntegrationID.ValueString(), plan.ProjectID.ValueString(), plan.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error granting integration project access",
			"Could not grant integration project access, unexpected error: "+err.Error(),
		)
		return
	}

	result := IntegrationProjectAccess{
		TeamID:        teamID,
		IntegrationID: plan.IntegrationID,
		ProjectID:     plan.ProjectID,
	}

	tflog.Info(ctx, "granted integration project access", map[string]interface{}{
		"team_id":        result.TeamID.ValueString(),
		"integration_id": result.IntegrationID.ValueString(),
		"project_id":     result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *integrationProjectAccessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state IntegrationProjectAccess
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID := state.TeamID
	if teamID.ValueString() == "" {
		t, err := r.client.Team(ctx, "")
		if err != nil {
			resp.Diagnostics.AddError(
				"Error getting team from provider",
				"Could not read team from provider, unexpected error: "+err.Error(),
			)
			return
		}
		teamID = types.StringValue(t.ID)
	}

	allowed, err := r.client.GetIntegrationProjectAccess(ctx, state.IntegrationID.ValueString(), state.ProjectID.ValueString(), state.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error granting integration project access",
			"Could not grant integration project access, unexpected error: "+err.Error(),
		)
		return
	}

	result := IntegrationProjectAccess{
		TeamID:        teamID,
		IntegrationID: state.IntegrationID,
		ProjectID:     state.ProjectID,
	}
	tflog.Info(ctx, "read integration project access", map[string]interface{}{
		"team_id":        result.TeamID.ValueString(),
		"integration_id": result.IntegrationID.ValueString(),
		"project_id":     result.ProjectID.ValueString(),
		"allowed":        allowed,
	})

	if allowed {
		diags = resp.State.Set(ctx, result)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		resp.State.RemoveResource(ctx)
	}
}

func (r *integrationProjectAccessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	panic("all plans should result in recreation")
}

func (r *integrationProjectAccessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan IntegrationProjectAccess
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID := plan.TeamID
	if teamID.ValueString() == "" {
		t, err := r.client.Team(ctx, "")
		if err != nil {
			resp.Diagnostics.AddError(
				"Error getting team from provider",
				"Could not read team from provider, unexpected error: "+err.Error(),
			)
			return
		}
		teamID = types.StringValue(t.ID)
	}

	allowed, err := r.client.RevokeIntegrationProjectAccess(ctx, plan.IntegrationID.ValueString(), plan.ProjectID.ValueString(), plan.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error revoking integration project access",
			"Could not revoke integration project access, unexpected error: "+err.Error(),
		)
		return
	}

	result := IntegrationProjectAccess{
		TeamID:        teamID,
		IntegrationID: plan.IntegrationID,
		ProjectID:     plan.ProjectID,
	}

	tflog.Info(ctx, "revoked integration project access", map[string]interface{}{
		"team_id":        result.TeamID.ValueString(),
		"integration_id": result.IntegrationID.ValueString(),
		"project_id":     result.ProjectID.ValueString(),
		"allowed":        allowed,
	})
}
