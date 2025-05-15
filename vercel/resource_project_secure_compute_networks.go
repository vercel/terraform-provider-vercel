package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

var (
	_ resource.Resource                = &projectSecureComputeNetworksResource{}
	_ resource.ResourceWithConfigure   = &projectSecureComputeNetworksResource{}
	_ resource.ResourceWithImportState = &projectSecureComputeNetworksResource{}
)

func newProjectSecureComputeNetworksResource() resource.Resource {
	return &projectSecureComputeNetworksResource{}
}

type projectSecureComputeNetworksResource struct {
	client *client.Client
}

func (r *projectSecureComputeNetworksResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_secure_compute_networks"
}

func (r *projectSecureComputeNetworksResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *projectSecureComputeNetworksResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: ` `,
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description:   "The ID of the Project",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Description:   "The team ID. Required when configuring a team resource if a default team has not been set in the provider.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"secure_compute_networks": schema.SetNestedAttribute{
				Description: "A set of Secure Compute Networks that the project should be configured with.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"environment": schema.StringAttribute{
							Description: "The environment being configured. Should be one of 'production', 'preview', or the ID of a Custom Environment",
							Required:    true,
						},
						"network_id": schema.StringAttribute{
							Description: "The ID of the Secure Compute Network to configure for this environment",
							Required:    true,
						},
						"passive": schema.BoolAttribute{
							Description: "Whether the Secure Compute Network should be configured as a passive network, meaning it is used for passive failover.",
							Required:    true,
						},
						"builds_enabled": schema.BoolAttribute{
							Description: "Whether the projects build container should be included in the Secure Compute Network.",
							Required:    true,
						},
					},
				},
				Validators: []validator.Set{
					NewPassiveBuildsEnabledValidator(),
				},
			},
		},
	}
}

type ProjectSecureComputeNetwork struct {
	Environment   types.String `tfsdk:"environment"`
	NetworkID     types.String `tfsdk:"network_id"`
	Passive       types.Bool   `tfsdk:"passive"`
	BuildsEnabled types.Bool   `tfsdk:"builds_enabled"`
}

var projectSecureComputeNetworkElemType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"environment":    types.StringType,
		"network_id":     types.StringType,
		"passive":        types.BoolType,
		"builds_enabled": types.BoolType,
	},
}

type ProjectSecureComputeNetworks struct {
	TeamID                types.String `tfsdk:"team_id"`
	ProjectID             types.String `tfsdk:"project_id"`
	SecureComputeNetworks types.Set    `tfsdk:"secure_compute_networks"`
}

func (p ProjectSecureComputeNetworks) toUpdateProjectSecureComputeNetworksRequest(ctx context.Context) (client.UpdateProjectSecureComputeNetworksRequest, diag.Diagnostics) {
	var networks []ProjectSecureComputeNetwork
	diags := p.SecureComputeNetworks.ElementsAs(ctx, &networks, false)
	scNetworks := make([]client.ConnectConfigurationRequest, 0, len(networks))
	for _, n := range networks {
		scNetworks = append(scNetworks, client.ConnectConfigurationRequest{
			Environment:            n.Environment.ValueString(),
			ConnectConfigurationID: n.NetworkID.ValueString(),
			Passive:                n.Passive.ValueBool(),
			BuildsEnabled:          n.BuildsEnabled.ValueBool(),
		})
	}
	return client.UpdateProjectSecureComputeNetworksRequest{
		TeamID:                p.TeamID.ValueString(),
		ProjectID:             p.ProjectID.ValueString(),
		SecureComputeNetworks: scNetworks,
	}, diags
}

func convertResponseToProjectSecureComputeNetworks(response client.ProjectResponse) ProjectSecureComputeNetworks {
	networks := make([]attr.Value, 0, len(response.ConnectConfigurations))
	for _, n := range response.ConnectConfigurations {
		networks = append(networks, types.ObjectValueMust(projectSecureComputeNetworkElemType.AttrTypes, map[string]attr.Value{
			"environment":    types.StringValue(n.Environment),
			"network_id":     types.StringValue(n.ConnectConfigurationID),
			"passive":        types.BoolValue(n.Passive),
			"builds_enabled": types.BoolValue(n.BuildsEnabled),
		}))
	}

	return ProjectSecureComputeNetworks{
		TeamID:                types.StringValue(response.TeamID),
		ProjectID:             types.StringValue(response.ID),
		SecureComputeNetworks: types.SetValueMust(projectSecureComputeNetworkElemType, networks),
	}
}

func (r *projectSecureComputeNetworksResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectSecureComputeNetworks
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError(
			"Error getting project secure compute networks plan",
			"Error getting project secure compute networks plan",
		)
		return
	}

	tflog.Info(ctx, "creating project secure compute networks", map[string]any{
		"team_id":    plan.TeamID.ValueString(),
		"project_id": plan.ProjectID.ValueString(),
	})

	request, diags := plan.toUpdateProjectSecureComputeNetworksRequest(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	out, err := r.client.UpdateProjectSecureComputeNetworks(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project secure compute networks",
			"Could not create project secure compute networks, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "created secure compute networks", map[string]any{
		"team_id":    plan.TeamID.ValueString(),
		"project_id": plan.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, convertResponseToProjectSecureComputeNetworks(out))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *projectSecureComputeNetworksResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectSecureComputeNetworks
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetProject(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading projject secure compute networks",
			fmt.Sprintf("Could not get project secure compute networks %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	diags = resp.State.Set(ctx, convertResponseToProjectSecureComputeNetworks(out))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *projectSecureComputeNetworksResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProjectSecureComputeNetworks
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError(
			"Error getting project secure compute networks plan",
			"Error getting project secure compute networks plan",
		)
		return
	}

	request, diags := plan.toUpdateProjectSecureComputeNetworksRequest(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	out, err := r.client.UpdateProjectSecureComputeNetworks(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project secure compute networks",
			"Could not update project secure compute networks, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "created secure compute networks", map[string]any{
		"team_id":    plan.TeamID.ValueString(),
		"project_id": plan.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, convertResponseToProjectSecureComputeNetworks(out))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *projectSecureComputeNetworksResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectSecureComputeNetworks
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "deleting secure compute networks", map[string]any{
		"project_id": state.ProjectID.ValueString(),
		"team_id":    state.TeamID.ValueString(),
	})

	_, err := r.client.UpdateProjectSecureComputeNetworks(ctx, client.UpdateProjectSecureComputeNetworksRequest{
		TeamID:                state.TeamID.ValueString(),
		ProjectID:             state.ProjectID.ValueString(),
		SecureComputeNetworks: nil,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project secure compute networks",
			"Could not delete project secure compute networks, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *projectSecureComputeNetworksResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Project Secure Compute Networks",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id\" or \"project_id\"", req.ID),
		)
	}
	out, err := r.client.GetProject(ctx, projectID, teamID)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project secure compute networks",
			fmt.Sprintf("Could not get project secure compute networks %s %s, unexpected error: %s",
				teamID,
				projectID,
				err,
			),
		)
		return
	}

	diags := resp.State.Set(ctx, convertResponseToProjectSecureComputeNetworks(out))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
