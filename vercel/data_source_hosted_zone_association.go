package vercel

import (
	"context"

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
func (h *hostedZoneAssociationDataSource) Configure(context.Context, datasource.ConfigureRequest, *datasource.ConfigureResponse) {
	panic("unimplemented")
}

func (h *hostedZoneAssociationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hosted_zone_association"
}

// Read implements datasource.DataSource.
func (h *hostedZoneAssociationDataSource) Read(context.Context, datasource.ReadRequest, *datasource.ReadResponse) {
	panic("unimplemented")
}

// Schema implements datasource.DataSource.
func (h *hostedZoneAssociationDataSource) Schema(context.Context, datasource.SchemaRequest, *datasource.SchemaResponse) {
	panic("unimplemented")
}
