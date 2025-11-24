package vercel

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &networkResource{}
	_ resource.ResourceWithConfigure   = &networkResource{}
	_ resource.ResourceWithImportState = &networkResource{}
)

func newNetworkResource() resource.Resource {
	return &networkResource{}
}

type networkResource struct {
	client *client.Client
}

func (r *networkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *networkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NetworkState

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Create(ctx, 15*time.Minute)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request := &client.CreateNetworkRequest{
		AWSAvailabilityZoneIDs: nil,
		CIDR:                   plan.CIDR.ValueString(),
		Name:                   plan.Name.ValueString(),
		Region:                 plan.Region.ValueString(),
		TeamID:                 plan.TeamID.ValueString(),
	}

	if !plan.AWSAvailabilityZoneIDs.IsNull() && !plan.AWSAvailabilityZoneIDs.IsUnknown() {
		var zoneIDs []string

		resp.Diagnostics.Append(plan.AWSAvailabilityZoneIDs.ElementsAs(ctx, &zoneIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		request.AWSAvailabilityZoneIDs = &zoneIDs
	}

	out, err := r.client.CreateNetwork(ctx, request)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Network",
			fmt.Sprintf("Could not create Network %s %s, unexpected error: %s",
				plan.TeamID.ValueString(),
				plan.Name.ValueString(),
				err,
			),
		)
		return
	}

	interval := 10 * time.Second
	attempts := max(1, int(timeout/interval))

poll:
	for attempt := range attempts {
		select {
		case <-ctx.Done():
			break poll
		default:
		}

		out, err = r.client.ReadNetwork(ctx, client.ReadNetworkRequest{
			NetworkID: out.ID,
			TeamID:    out.TeamID,
		})

		if err != nil || out.Status == "ready" {
			break
		}

		tflog.Info(ctx, "Still creating...", map[string]any{
			"id":      out.ID,
			"team_id": out.TeamID,
		})

		if attempt < attempts-1 {
			time.Sleep(interval)
		}
	}

	if out.Status != "ready" {
		resp.Diagnostics.AddError(
			"Error waiting for Network to be ready",
			fmt.Sprintf("Network status is %s after %d attempts", out.Status, attempts),
		)
		return
	}

	result, diags := toNetworkState(ctx, out)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Created Network", map[string]any{
		"aws_account_id": result.AWSAccountID.ValueString(),
		"aws_region":     result.AWSRegion.ValueString(),
		"cidr":           result.CIDR.ValueString(),
		"id":             result.ID.ValueString(),
		"name":           result.Name.ValueString(),
		"region":         result.Region.ValueString(),
		"status":         result.Status.ValueString(),
		"team_id":        result.TeamID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *networkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NetworkState

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteNetwork(ctx, client.DeleteNetworkRequest{
		NetworkID: state.ID.ValueString(),
		TeamID:    state.TeamID.ValueString(),
	})

	if client.NotFound(err) {
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Network",
			fmt.Sprintf("Could not delete Network %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "Deleted Network", map[string]any{
		"aws_account_id": state.AWSAccountID.ValueString(),
		"aws_region":     state.AWSRegion.ValueString(),
		"cidr":           state.CIDR.ValueString(),
		"id":             state.ID.ValueString(),
		"name":           state.Name.ValueString(),
		"region":         state.Region.ValueString(),
		"status":         state.Status.ValueString(),
		"team_id":        state.TeamID.ValueString(),
	})
}

func (r *networkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamIDOrEmpty, networkID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid ID specified",
			fmt.Sprintf("Invalid ID '%s' specified. It should match the following format \"network_id\" or \"team_id/network_id\"", req.ID),
		)
		return
	}

	out, err := r.client.ReadNetwork(ctx, client.ReadNetworkRequest{
		NetworkID: networkID,
		TeamID:    teamIDOrEmpty,
	})

	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Network",
			fmt.Sprintf("Could not read Network %s %s, unexpected error: %s",
				teamIDOrEmpty,
				networkID,
				err,
			),
		)
		return
	}

	result, diags := toNetworkState(ctx, out)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Read Network", map[string]any{
		"aws_account_id": result.AWSAccountID.ValueString(),
		"aws_region":     result.AWSRegion.ValueString(),
		"cidr":           result.CIDR.ValueString(),
		"id":             result.ID.ValueString(),
		"name":           result.Name.ValueString(),
		"region":         result.Region.ValueString(),
		"status":         result.Status.ValueString(),
		"team_id":        result.TeamID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *networkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (r *networkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NetworkState

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.ReadNetwork(ctx, client.ReadNetworkRequest{
		NetworkID: state.ID.ValueString(),
		TeamID:    state.TeamID.ValueString(),
	})

	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Network",
			fmt.Sprintf("Could not read Network %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := toNetworkState(ctx, out)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Read Network", map[string]any{
		"aws_account_id": result.AWSAccountID.ValueString(),
		"aws_region":     result.AWSRegion.ValueString(),
		"cidr":           result.CIDR.ValueString(),
		"id":             result.ID.ValueString(),
		"name":           result.Name.ValueString(),
		"region":         result.Region.ValueString(),
		"status":         result.Status.ValueString(),
		"team_id":        result.TeamID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *networkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a Network resource.",
		Attributes: map[string]schema.Attribute{
			"aws_account_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the AWS Account in which the network exists.",
			},
			"aws_availability_zone_ids": schema.ListAttribute{
				Computed:      true,
				Description:   "The IDs of the AWS Availability Zones in which the network exists, if specified during creation.",
				ElementType:   types.StringType,
				Optional:      true,
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"aws_region": schema.StringAttribute{
				Computed:    true,
				Description: "The AWS Region in which the network exists.",
			},
			"cidr": schema.StringAttribute{
				Description:   "The CIDR range of the Network.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Required:      true,
			},
			"egress_ip_addresses": schema.ListAttribute{
				Computed:    true,
				Description: "The egress IP addresses of the Network.",
				ElementType: types.StringType,
			},
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The unique identifier of the Network.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the network.",
				Required:    true,
			},
			"region": schema.StringAttribute{
				Description:   "The Vercel region in which the Network exists.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Required:      true,
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "The status of the Network.",
			},
			"team_id": schema.StringAttribute{
				Computed:      true,
				Description:   "The unique identifier of the Team that owns the Network. Required when configuring a team resource if a default team has not been set in the provider.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
			}),
			"vpc_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the AWS VPC which hosts the network.",
			},
		},
	}
}

func (r *networkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NetworkState

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateNetwork(ctx, client.UpdateNetworkRequest{
		NetworkID: plan.ID.ValueString(),
		Name:      plan.Name.ValueString(),
		TeamID:    plan.TeamID.ValueString(),
	})

	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Network",
			fmt.Sprintf("Could not update Network %s %s, unexpected error: %s",
				plan.TeamID.ValueString(),
				plan.ID.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := toNetworkState(ctx, out)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Updated Network", map[string]any{
		"aws_account_id": result.AWSAccountID.ValueString(),
		"aws_region":     result.AWSRegion.ValueString(),
		"cidr":           result.CIDR.ValueString(),
		"id":             result.ID.ValueString(),
		"name":           result.Name.ValueString(),
		"region":         result.Region.ValueString(),
		"status":         result.Status.ValueString(),
		"team_id":        result.TeamID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}
