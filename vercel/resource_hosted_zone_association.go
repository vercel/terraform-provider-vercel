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
	_ resource.Resource                = &hostedZoneAssociationResource{}
	_ resource.ResourceWithConfigure   = &hostedZoneAssociationResource{}
	_ resource.ResourceWithImportState = &hostedZoneAssociationResource{}
)

func newHostedZoneAssociationResource() resource.Resource {
	return &hostedZoneAssociationResource{}
}

type hostedZoneAssociationResource struct {
	client *client.Client
}

type HostedZoneAssociationState struct {
	ConfigurationID types.String `tfsdk:"configuration_id"`
	HostedZoneID    types.String `tfsdk:"hosted_zone_id"`
	HostedZoneName  types.String `tfsdk:"hosted_zone_name"`
	Owner           types.String `tfsdk:"owner"`
	TeamID          types.String `tfsdk:"team_id"`
}

func (r *hostedZoneAssociationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *hostedZoneAssociationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan HostedZoneAssociationState

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateHostedZoneAssociation(ctx, client.CreateHostedZoneAssociationRequest{
		ConfigurationID: plan.ConfigurationID.ValueString(),
		HostedZoneID:    plan.HostedZoneID.ValueString(),
		TeamID:          plan.TeamID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Hosted Zone Association",
			fmt.Sprintf("Could not create Hosted Zone Association %s %s, unexpected error: %s",
				plan.ConfigurationID.ValueString(),
				plan.HostedZoneID.ValueString(),
				err,
			),
		)
		return
	}

	result := HostedZoneAssociationState{
		ConfigurationID: types.StringValue(out.ConfigurationID),
		HostedZoneID:    types.StringValue(out.HostedZoneID),
		HostedZoneName:  types.StringValue(""), // Will be populated on next read
		Owner:           types.StringValue(""), // Will be populated on next read
		TeamID:          toTeamID(plan.TeamID.ValueString()),
	}

	// The create endpoint, unlike other verbs, only returns the
	// `configurationId` and `hostedZoneId` fields. We need to make a
	// follow-up read to get the complete information.
	association, err := r.client.GetHostedZoneAssociation(ctx, client.GetHostedZoneAssociationRequest{
		ConfigurationID: out.ConfigurationID,
		HostedZoneID:    out.HostedZoneID,
		TeamID:          plan.TeamID.ValueString(),
	})

	if err != nil {
		// If the read fails after successful creation, we'll proceed with
		// partial information to at least register the resource into state.
		// Subsequent read operations will populate the missing fields.
		tflog.Warn(ctx, "Could not read complete Hosted Zone Association data after creation, proceeding with partial state", map[string]any{
			"configuration_id": out.ConfigurationID,
			"hosted_zone_id":   out.HostedZoneID,
			"error":            err.Error(),
		})
	} else {
		result.HostedZoneName = types.StringValue(association.HostedZoneName)
		result.Owner = types.StringValue(association.Owner)
	}

	tflog.Info(ctx, "Created Hosted Zone Association", map[string]any{
		"configuration_id": result.ConfigurationID.ValueString(),
		"hosted_zone_id":   result.HostedZoneID.ValueString(),
		"hosted_zone_name": result.HostedZoneName.ValueString(),
		"owner":            result.Owner.ValueString(),
		"team_id":          result.TeamID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *hostedZoneAssociationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state HostedZoneAssociationState

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteHostedZoneAssociation(ctx, client.DeleteHostedZoneAssociationRequest{
		ConfigurationID: state.ConfigurationID.ValueString(),
		HostedZoneID:    state.HostedZoneID.ValueString(),
		TeamID:          state.TeamID.ValueString(),
	})

	if client.NotFound(err) {
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Hosted Zone Association",
			fmt.Sprintf("Could not delete Hosted Zone Association %s %s, unexpected error: %s",
				state.ConfigurationID.ValueString(),
				state.HostedZoneID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "Deleted Hosted Zone Association", map[string]any{
		"configuration_id": state.ConfigurationID.ValueString(),
		"hosted_zone_id":   state.HostedZoneID.ValueString(),
		"hosted_zone_name": state.HostedZoneName.ValueString(),
		"owner":            state.Owner.ValueString(),
		"team_id":          state.TeamID.ValueString(),
	})
}

func (r *hostedZoneAssociationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamIDOrEmpty, configurationID, hostedZoneID, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid ID specified",
			fmt.Sprintf("Invalid ID '%s' specified. It should match the following format \"configuration_id/hosted_zone_id\" or \"team_id/configuration_id/hosted_zone_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetHostedZoneAssociation(ctx, client.GetHostedZoneAssociationRequest{
		ConfigurationID: configurationID,
		HostedZoneID:    hostedZoneID,
		TeamID:          teamIDOrEmpty,
	})

	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Hosted Zone Association",
			fmt.Sprintf("Could not read Hosted Zone Association %s %s, unexpected error: %s",
				configurationID,
				hostedZoneID,
				err,
			),
		)
		return
	}

	result := HostedZoneAssociationState{
		ConfigurationID: types.StringValue(configurationID),
		HostedZoneID:    types.StringValue(out.HostedZoneID),
		HostedZoneName:  types.StringValue(out.HostedZoneName),
		Owner:           types.StringValue(out.Owner),
		TeamID:          toTeamID(teamIDOrEmpty),
	}

	tflog.Info(ctx, "Read Hosted Zone Association", map[string]any{
		"configuration_id": result.ConfigurationID.ValueString(),
		"hosted_zone_id":   result.HostedZoneID.ValueString(),
		"hosted_zone_name": result.HostedZoneName.ValueString(),
		"owner":            result.Owner.ValueString(),
		"team_id":          result.TeamID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *hostedZoneAssociationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hosted_zone_association"
}

func (r *hostedZoneAssociationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state HostedZoneAssociationState

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetHostedZoneAssociation(ctx, client.GetHostedZoneAssociationRequest{
		ConfigurationID: state.ConfigurationID.ValueString(),
		HostedZoneID:    state.HostedZoneID.ValueString(),
		TeamID:          state.TeamID.ValueString(),
	})

	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Hosted Zone Association",
			fmt.Sprintf("Could not read Hosted Zone Association %s %s, unexpected error: %s",
				state.ConfigurationID.ValueString(),
				state.HostedZoneID.ValueString(),
				err,
			),
		)
		return
	}

	result := HostedZoneAssociationState{
		ConfigurationID: types.StringValue(state.ConfigurationID.ValueString()),
		HostedZoneID:    types.StringValue(out.HostedZoneID),
		HostedZoneName:  types.StringValue(out.HostedZoneName),
		Owner:           types.StringValue(out.Owner),
		TeamID:          toTeamID(state.TeamID.ValueString()),
	}

	tflog.Info(ctx, "Read Hosted Zone Association", map[string]any{
		"configuration_id": result.ConfigurationID.ValueString(),
		"hosted_zone_id":   result.HostedZoneID.ValueString(),
		"hosted_zone_name": result.HostedZoneName.ValueString(),
		"owner":            result.Owner.ValueString(),
		"team_id":          result.TeamID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *hostedZoneAssociationResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Hosted Zone Association resource.

Hosted Zone Associations provide a way to associate an AWS Route53 Hosted Zone with a Secure Compute network.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs).
`,
		Attributes: map[string]schema.Attribute{
			"configuration_id": schema.StringAttribute{
				Description:   "The ID of the Secure Compute network to associate the Hosted Zone with.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"hosted_zone_id": schema.StringAttribute{
				Description:   "The ID of the Hosted Zone to associate.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"hosted_zone_name": schema.StringAttribute{
				Description: "The name of the Hosted Zone.",
				Computed:    true,
			},
			"owner": schema.StringAttribute{
				Description: "The ID of the AWS Account that owns the Hosted Zone.",
				Computed:    true,
			},
			"team_id": schema.StringAttribute{
				Description:   "The ID of the team the Hosted Zone Association should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *hostedZoneAssociationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All changes to this resource should require recreation, as the
	// underlying API does not expose a method to perform in-place updates
	// (not that it should).
	//
	// This function should never be called since all schema properties are
	// annotated with the `RequiresReplace` plan modifier, but we need to
	// implement it regardless to satisfy the interface.
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"The Hosted Zone Association resource does not support in-place updates. All changes require recreation of the resource.",
	)
}
