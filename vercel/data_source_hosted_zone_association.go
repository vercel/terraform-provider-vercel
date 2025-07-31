package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &hostedZoneAssociationDataSource{}
	_ datasource.DataSourceWithConfigure = &hostedZoneAssociationDataSource{}
)

func newHostedZoneAssociationDataSource() datasource.DataSource {
	return &hostedZoneAssociationDataSource{}
}

type hostedZoneAssociationDataSource struct {
	client *client.Client
}

// Configure implements datasource.DataSourceWithConfigure.
func (r *hostedZoneAssociationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *hostedZoneAssociationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hosted_zone_association"
}

// Read implements datasource.DataSource.
func (r *hostedZoneAssociationDataSource) Read(context.Context, datasource.ReadRequest, *datasource.ReadResponse) {
	panic("unimplemented")
}

func (r *hostedZoneAssociationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Hosted Zone Association.

Hosted Zone Associations provide a way to associate an AWS Route53 Hosted Zone with a Secure Compute network.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs).
`,
		Attributes: map[string]schema.Attribute{
			"configuration_id": schema.StringAttribute{
				Description: "The ID of the Secure Compute network to associate the Hosted Zone with.",
				Required:    true,
			},
			"hosted_zone_id": schema.StringAttribute{
				Description: "The ID of the Hosted Zone to associate.",
				Required:    true,
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
				Description: "The ID of the team the Hosted Zone Association should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}
