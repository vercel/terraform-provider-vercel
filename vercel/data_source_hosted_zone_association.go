package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
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

// Schema implements datasource.DataSource.
func (r *hostedZoneAssociationDataSource) Schema(context.Context, datasource.SchemaRequest, *datasource.SchemaResponse) {
	panic("unimplemented")
}
