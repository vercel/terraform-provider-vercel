package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &networkDataSource{}
	_ datasource.DataSourceWithConfigure = &networkDataSource{}
)

func newHostedZoneAssociationDataSource() datasource.DataSource {
	return &networkDataSource{}
}

type networkDataSource struct {
	client *client.Client
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

func (n *networkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

// Read implements datasource.DataSourceWithConfigure.
func (n *networkDataSource) Read(context.Context, datasource.ReadRequest, *datasource.ReadResponse) {
	panic("unimplemented")
}

// Schema implements datasource.DataSourceWithConfigure.
func (n *networkDataSource) Schema(context.Context, datasource.SchemaRequest, *datasource.SchemaResponse) {
	panic("unimplemented")
}
