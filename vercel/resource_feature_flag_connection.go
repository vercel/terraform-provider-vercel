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

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                = &featureFlagConnectionResource{}
	_ resource.ResourceWithConfigure   = &featureFlagConnectionResource{}
	_ resource.ResourceWithImportState = &featureFlagConnectionResource{}
)

func newFeatureFlagConnectionResource() resource.Resource {
	return &featureFlagConnectionResource{}
}

type featureFlagConnectionResource struct {
	client *client.Client
}

type featureFlagConnectionModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectID         types.String `tfsdk:"project_id"`
	ConsumerProjectID types.String `tfsdk:"consumer_project_id"`
	TeamID            types.String `tfsdk:"team_id"`
}

func (r *featureFlagConnectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature_flag_connection"
}

func (r *featureFlagConnectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *featureFlagConnectionResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Feature Flag Connection resource.

A connection lets a consumer project in the same team read the flags of a source project. The consumer reads the source project's flags with its own deployment identity by naming the source project in the Flags SDK. Deleting the connection revokes access immediately, without a redeploy.

Vercel Flags must be enabled on the source project before other projects can be connected to it.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the connection, in the format `project_id/consumer_project_id`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the source Vercel project whose flags are shared.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"consumer_project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the Vercel project in the same team that may read the flags of the source project.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the Vercel team. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
		},
	}
}

func (r *featureFlagConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan featureFlagConnectionModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.TeamID = types.StringValue(r.client.TeamID(plan.TeamID.ValueString()))

	out, err := r.client.CreateFeatureFlagConnection(ctx, client.CreateFeatureFlagConnectionRequest{
		ProjectID:         plan.ProjectID.ValueString(),
		TeamID:            plan.TeamID.ValueString(),
		ConsumerProjectID: plan.ConsumerProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Feature Flag Connection",
			"Could not create Feature Flag Connection, unexpected error: "+err.Error(),
		)
		return
	}

	result := featureFlagConnectionFromClient(out, plan.TeamID)
	tflog.Info(ctx, "created feature flag connection", map[string]any{
		"project_id":          result.ProjectID.ValueString(),
		"consumer_project_id": result.ConsumerProjectID.ValueString(),
		"team_id":             result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *featureFlagConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state featureFlagConnectionModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.TeamID = types.StringValue(r.client.TeamID(state.TeamID.ValueString()))

	out, found, err := r.findConnection(ctx, state.TeamID.ValueString(), state.ProjectID.ValueString(), state.ConsumerProjectID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Feature Flag Connection",
			fmt.Sprintf(
				"Could not read Feature Flag Connection %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ProjectID.ValueString(),
				state.ConsumerProjectID.ValueString(),
				err,
			),
		)
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	diags = resp.State.Set(ctx, featureFlagConnectionFromClient(out, state.TeamID))
	resp.Diagnostics.Append(diags...)
}

func (r *featureFlagConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Updating a Feature Flag Connection is not supported",
		"Updating a Feature Flag Connection is not supported. Change any configurable field to force recreation.",
	)
}

func (r *featureFlagConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state featureFlagConnectionModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.TeamID = types.StringValue(r.client.TeamID(state.TeamID.ValueString()))

	err := r.client.DeleteFeatureFlagConnection(ctx, client.DeleteFeatureFlagConnectionRequest{
		ProjectID:         state.ProjectID.ValueString(),
		ConsumerProjectID: state.ConsumerProjectID.ValueString(),
		TeamID:            state.TeamID.ValueString(),
	})
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Feature Flag Connection",
			fmt.Sprintf(
				"Could not delete Feature Flag Connection %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ProjectID.ValueString(),
				state.ConsumerProjectID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted feature flag connection", map[string]any{
		"project_id":          state.ProjectID.ValueString(),
		"consumer_project_id": state.ConsumerProjectID.ValueString(),
		"team_id":             state.TeamID.ValueString(),
	})
}

func (r *featureFlagConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, consumerProjectID, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Feature Flag Connection",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id/consumer_project_id\" or \"project_id/consumer_project_id\"", req.ID),
		)
		return
	}
	teamID = r.client.TeamID(teamID)

	out, found, err := r.findConnection(ctx, teamID, projectID, consumerProjectID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing Feature Flag Connection",
			fmt.Sprintf("Could not read Feature Flag Connection %s %s %s, unexpected error: %s", teamID, projectID, consumerProjectID, err),
		)
		return
	}
	if !found {
		resp.Diagnostics.AddError(
			"Error importing Feature Flag Connection",
			fmt.Sprintf("Feature Flag Connection %s %s %s not found", teamID, projectID, consumerProjectID),
		)
		return
	}

	diags := resp.State.Set(ctx, featureFlagConnectionFromClient(out, types.StringValue(teamID)))
	resp.Diagnostics.Append(diags...)
}

func (r *featureFlagConnectionResource) findConnection(ctx context.Context, teamID, projectID, consumerProjectID string) (client.FeatureFlagConnection, bool, error) {
	connections, err := r.client.ListFeatureFlagConnections(ctx, client.ListFeatureFlagConnectionsRequest{
		ProjectID: projectID,
		TeamID:    teamID,
	})
	if err != nil {
		return client.FeatureFlagConnection{}, false, err
	}
	for _, connection := range connections {
		if connection.ConsumerProjectID == consumerProjectID {
			return connection, true, nil
		}
	}
	return client.FeatureFlagConnection{}, false, nil
}

func featureFlagConnectionFromClient(out client.FeatureFlagConnection, teamID types.String) featureFlagConnectionModel {
	return featureFlagConnectionModel{
		ID:                types.StringValue(fmt.Sprintf("%s/%s", out.SourceProjectID, out.ConsumerProjectID)),
		ProjectID:         types.StringValue(out.SourceProjectID),
		ConsumerProjectID: types.StringValue(out.ConsumerProjectID),
		TeamID:            teamID,
	}
}
