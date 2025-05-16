package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

var (
	_ datasource.DataSource              = &secureComputeNetworkDataSource{}
	_ datasource.DataSourceWithConfigure = &secureComputeNetworkDataSource{}
)

func newSecureComputeNetworkDataSource() datasource.DataSource {
	return &secureComputeNetworkDataSource{}
}

type secureComputeNetworkDataSource struct {
	client *client.Client
}

func (d *secureComputeNetworkDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secure_compute_network"
}

func (d *secureComputeNetworkDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Schema returns the schema information for an secureComputeNetwork data source
func (r *secureComputeNetworkDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Vercel Secure Compute Network.

This data source allows you to retrieve details about a Secure Compute Network by its name and optional team ID.
`,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the Secure Compute Network configuration.",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Description: "The ID of the Vercel team the Secure Compute Network belongs to. " +
					"If omitted, the provider will use the team configured on the provider or the user's default team.",
				Optional: true,
			},
			"id": schema.StringAttribute{
				Description: "The unique identifier of the Secure Compute Network.",
				Computed:    true,
			},
			"dc": schema.StringAttribute{
				Description: "The data center (region) associated with the Secure Compute Network.",
				Computed:    true,
			},
			"project_ids": schema.SetAttribute{
				Description: "A list of Vercel Project IDs connected to this Secure Compute Network.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"projects_count": schema.Int64Attribute{
				Description: "The number of Vercel Projects connected to this Secure Compute Network.",
				Computed:    true,
			},
			"peering_connections_count": schema.Int64Attribute{
				Description: "The number of peering connections established for this Secure Compute Network.",
				Computed:    true,
			},
			"cidr_block": schema.StringAttribute{
				Description: "The CIDR block assigned to the Secure Compute Network.",
				Computed:    true,
			},
			"availability_zone_ids": schema.SetAttribute{
				Description: "A set of AWS Availability Zone IDs where the Secure Compute Network resources are deployed.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"version": schema.StringAttribute{
				Description: "The current version identifier of the Secure Compute Network configuration.",
				Computed:    true,
			},
			"configuration_status": schema.StringAttribute{
				Description: "The operational status of the Secure Compute Network (e.g., 'ready', 'create_in_progress').",
				Computed:    true,
			},
			"aws": schema.SingleNestedAttribute{
				Description: "AWS configuration for the Secure Compute Network.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"account_id": schema.StringAttribute{
						Description: "The AWS account ID.",
						Computed:    true,
					},
					"region": schema.StringAttribute{
						Description: "The AWS region.",
						Computed:    true,
					},
					"elastic_ip_addresses": schema.SetAttribute{
						Description: "A list of Elastic IP addresses.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"lambda_role_arn": schema.StringAttribute{
						Description: "The ARN of the Lambda role.",
						Computed:    true,
					},
					"security_group_id": schema.StringAttribute{
						Description: "The ID of the security group.",
						Computed:    true,
					},
					"stack_id": schema.StringAttribute{
						Description: "The ID of the CloudFormation stack.",
						Computed:    true,
					},
					"subnet_ids": schema.SetAttribute{
						Description: "A list of subnet IDs.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"subscription_arn": schema.StringAttribute{
						Description: "The ARN of the subscription.",
						Computed:    true,
					},
					"vpc_id": schema.StringAttribute{
						Description: "The ID of the VPC.",
						Computed:    true,
					},
				},
			},
		},
	}
}

type SecureComputeNetworkAWS struct {
	AccountID          types.String `tfsdk:"account_id"`
	Region             types.String `tfsdk:"region"`
	ElasticIpAddresses types.Set    `tfsdk:"elastic_ip_addresses"`
	LambdaRoleArn      types.String `tfsdk:"lambda_role_arn"`
	SecurityGroupID    types.String `tfsdk:"security_group_id"`
	StackID            types.String `tfsdk:"stack_id"`
	SubnetIDs          types.Set    `tfsdk:"subnet_ids"`
	SubscriptionArn    types.String `tfsdk:"subscription_arn"`
	VPCID              types.String `tfsdk:"vpc_id"`
}

var secureComputeNetworkAWSAttrTypes = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"account_id": types.StringType,
		"region":     types.StringType,
		"elastic_ip_addresses": types.SetType{
			ElemType: types.StringType,
		},
		"lambda_role_arn":   types.StringType,
		"security_group_id": types.StringType,
		"stack_id":          types.StringType,
		"subnet_ids": types.SetType{
			ElemType: types.StringType,
		},
		"subscription_arn": types.StringType,
		"vpc_id":           types.StringType,
	},
}

type SecureComputeNetwork struct {
	AvailabilityZoneIDs     types.Set    `tfsdk:"availability_zone_ids"`
	CIDRBlock               types.String `tfsdk:"cidr_block"`
	ConfigurationStatus     types.String `tfsdk:"configuration_status"`
	DC                      types.String `tfsdk:"dc"`
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	PeeringConnectionsCount types.Int64  `tfsdk:"peering_connections_count"`
	ProjectIDs              types.Set    `tfsdk:"project_ids"`
	ProjectsCount           types.Int64  `tfsdk:"projects_count"`
	TeamID                  types.String `tfsdk:"team_id"`
	Version                 types.String `tfsdk:"version"`
	AWS                     types.Object `tfsdk:"aws"`
}

func convertResponseToSecureComputeNetwork(ctx context.Context, response *client.SecureComputeNetwork) (out SecureComputeNetwork, diags diag.Diagnostics) {
	projectIDs, ds := stringsToSet(ctx, response.ProjectIDs)
	diags.Append(ds...)
	if diags.HasError() {
		return out, diags
	}

	azIDs, ds := stringsToSet(ctx, response.AvailabilityZoneIDs)
	diags.Append(ds...)
	if diags.HasError() {
		return SecureComputeNetwork{}, diags
	}

	aws, ds := convertResponseToSecureComputeNetworkAWS(ctx, response.AWS)
	diags.Append(ds...)
	if diags.HasError() {
		return SecureComputeNetwork{}, diags
	}

	return SecureComputeNetwork{
		AvailabilityZoneIDs:     azIDs,
		CIDRBlock:               types.StringPointerValue(response.CIDRBlock),
		ConfigurationStatus:     types.StringValue(response.ConfigurationStatus),
		DC:                      types.StringValue(response.DC),
		ID:                      types.StringValue(response.ID),
		Name:                    types.StringValue(response.ConfigurationName),
		PeeringConnectionsCount: types.Int64PointerValue(intPtrToInt64Ptr(response.PeeringConnectionsCount)),
		ProjectIDs:              projectIDs,
		ProjectsCount:           types.Int64PointerValue(intPtrToInt64Ptr(response.ProjectsCount)),
		TeamID:                  types.StringValue(response.TeamID),
		Version:                 types.StringValue(response.Version),
		AWS:                     aws,
	}, diags
}

func convertResponseToSecureComputeNetworkAWS(ctx context.Context, aws client.SecureComputeNetworkAWS) (basetypes.ObjectValue, diag.Diagnostics) {
	elasticIpAddresses, diags := stringsToSet(ctx, aws.ElasticIpAddresses)
	if diags.HasError() {
		return types.ObjectNull(secureComputeNetworkAWSAttrTypes.AttrTypes), diags
	}

	subnetIDs, diags := stringsToSet(ctx, aws.SubnetIds)
	if diags.HasError() {
		return types.ObjectNull(secureComputeNetworkAWSAttrTypes.AttrTypes), diags
	}

	return types.ObjectValueMust(
		secureComputeNetworkAWSAttrTypes.AttrTypes, map[string]attr.Value{
			"account_id":           types.StringValue(aws.AccountID),
			"region":               types.StringValue(aws.Region),
			"elastic_ip_addresses": elasticIpAddresses,
			"lambda_role_arn":      types.StringPointerValue(aws.LambdaRoleArn),
			"security_group_id":    types.StringPointerValue(aws.SecurityGroupId),
			"stack_id":             types.StringPointerValue(aws.StackId),
			"subnet_ids":           subnetIDs,
			"subscription_arn":     types.StringPointerValue(aws.SubscriptionArn),
			"vpc_id":               types.StringPointerValue(aws.VpcId),
		}), diags
}

func (d *secureComputeNetworkDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var config SecureComputeNetwork
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading Secure Compute Network data source", map[string]any{"name": config.Name.ValueString(), "team_id": config.TeamID.ValueString()})

	networks, err := d.client.ListSecureComputeNetworks(ctx, config.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"Unable to list Secure Compute Networks: %s",
				err,
			),
		)
		return
	}

	var network *client.SecureComputeNetwork
	for i, n := range networks {
		if n.ConfigurationName == config.Name.ValueString() {
			network = &networks[i]
			break
		}
	}

	if network == nil {
		resp.Diagnostics.AddError("Secure Compute Network Not Found", fmt.Sprintf(
			"No Secure Compute Network found with name '%s' for team_id '%s'",
			config.Name.ValueString(),
			d.client.TeamID(config.TeamID.ValueString()),
		))
		return
	}

	out, diags := convertResponseToSecureComputeNetwork(ctx, network)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &out)
	resp.Diagnostics.Append(diags...)
}
