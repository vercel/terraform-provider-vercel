package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &networkDataSource{}
	_ datasource.DataSourceWithConfigure = &networkDataSource{}
)

func newNetworkDataSource() datasource.DataSource {
	return &networkDataSource{}
}

type networkDataSource struct {
	client *client.Client
}

type NetworkDataSource struct {
	AWSAccountID           types.String `tfsdk:"aws_account_id"`
	AWSAvailabilityZoneIDs types.List   `tfsdk:"aws_availability_zone_ids"`
	AWSRegion              types.String `tfsdk:"aws_region"`
	CIDR                   types.String `tfsdk:"cidr"`
	EgressIPAddresses      types.List   `tfsdk:"egress_ip_addresses"`
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Region                 types.String `tfsdk:"region"`
	Status                 types.String `tfsdk:"status"`
	VPCID                  types.String `tfsdk:"vpc_id"`
	TeamID                 types.String `tfsdk:"team_id"`
}

func (r *networkDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *networkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (r *networkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state NetworkDataSource

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
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

	result, diags := ToNetworkDataSource(ctx, out)
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

func (r *networkDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about a Vercel Network.",
		Attributes: map[string]schema.Attribute{
			"aws_account_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the AWS Account in which the network exists.",
			},
			"aws_availability_zone_ids": schema.ListAttribute{
				Computed:    true,
				Description: "The IDs of the AWS Availability Zones in which the network exists, if specified during creation.",
				ElementType: types.StringType,
			},
			"aws_region": schema.StringAttribute{
				Computed:    true,
				Description: "The AWS Region in which the network exists.",
			},
			"cidr": schema.StringAttribute{
				Computed:    true,
				Description: "The CIDR range of the Network.",
			},
			"egress_ip_addresses": schema.ListAttribute{
				Computed:    true,
				Description: "The egress IP addresses of the Network.",
				ElementType: types.StringType,
			},
			"id": schema.StringAttribute{
				Description: "The unique identifier of the Network.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the network.",
			},
			"region": schema.StringAttribute{
				Computed:    true,
				Description: "The Vercel region in which the Network exists.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "The status of the Network.",
			},
			"team_id": schema.StringAttribute{
				Description: "The unique identifier of the Team that owns the Network. Required when configuring a team resource if a default team has not been set in the provider.",
				Optional:    true,
			},
			"vpc_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the AWS VPC which hosts the network.",
			},
		},
	}
}

func ToNetworkDataSource(ctx context.Context, network client.Network) (NetworkDataSource, diag.Diagnostics) {
	var diags diag.Diagnostics

	state := NetworkDataSource{
		AWSAccountID:           types.StringValue(network.AWSAccountID),
		AWSAvailabilityZoneIDs: types.ListNull(types.StringType),
		AWSRegion:              types.StringValue(network.AWSRegion),
		CIDR:                   types.StringValue(network.CIDR),
		EgressIPAddresses:      types.ListNull(types.StringType),
		ID:                     types.StringValue(network.ID),
		Name:                   types.StringValue(network.Name),
		Region:                 types.StringValue(network.Region),
		Status:                 types.StringValue(network.Status),
		TeamID:                 toTeamID(network.TeamID),
		VPCID:                  types.StringNull(),
	}

	if network.AWSAvailabilityZoneIDs != nil {
		list, listDiags := types.ListValueFrom(ctx, types.StringType, *network.AWSAvailabilityZoneIDs)
		diags.Append(listDiags...)
		if !diags.HasError() {
			state.AWSAvailabilityZoneIDs = list
		}
	}

	if network.EgressIPAddresses != nil {
		list, listDiags := types.ListValueFrom(ctx, types.StringType, *network.EgressIPAddresses)
		diags.Append(listDiags...)
		if !diags.HasError() {
			state.EgressIPAddresses = list
		}
	}

	if network.VPCID != nil {
		state.VPCID = types.StringValue(*network.VPCID)
	}

	return state, diags
}
