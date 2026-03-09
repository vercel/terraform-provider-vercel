package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var (
	_ datasource.DataSource              = &blobStoresDataSource{}
	_ datasource.DataSourceWithConfigure = &blobStoresDataSource{}
)

func newBlobStoresDataSource() datasource.DataSource {
	return &blobStoresDataSource{}
}

type blobStoresDataSource struct {
	client *client.Client
}

type BlobStoresDataSourceModel struct {
	Stores []BlobStoreListItem `tfsdk:"stores"`
	TeamID types.String        `tfsdk:"team_id"`
}

func (d *blobStoresDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blob_stores"
}

func (d *blobStoresDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *blobStoresDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides the Blob stores available to the configured team or personal account.",
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team whose Blob stores should be listed. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"stores": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The Blob stores available to the configured team or personal account.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The ID of the Blob store.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the Blob store.",
						},
						"access": schema.StringAttribute{
							Computed:    true,
							Description: "Whether blobs created in this store are `public` or `private` by default.",
						},
						"region": schema.StringAttribute{
							Computed:    true,
							Description: "The region in which the Blob store exists.",
						},
						"status": schema.StringAttribute{
							Computed:    true,
							Description: "The current status of the Blob store.",
						},
						"size": schema.Int64Attribute{
							Computed:    true,
							Description: "The size of the Blob store in bytes.",
						},
						"file_count": schema.Int64Attribute{
							Computed:    true,
							Description: "The number of files currently stored in the Blob store.",
						},
						"created_at": schema.Int64Attribute{
							Computed:    true,
							Description: "The Unix timestamp, in milliseconds, when the Blob store was created.",
						},
						"updated_at": schema.Int64Attribute{
							Computed:    true,
							Description: "The Unix timestamp, in milliseconds, when the Blob store was last updated.",
						},
					},
				},
			},
		},
	}
}

func (d *blobStoresDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config BlobStoresDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stores, err := d.client.ListBlobStores(ctx, config.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Blob stores",
			fmt.Sprintf("Could not read Blob stores for team %s, unexpected error: %s", config.TeamID.ValueString(), err),
		)
		return
	}

	sortBlobStores(stores)
	result := BlobStoresDataSourceModel{
		Stores: make([]BlobStoreListItem, 0, len(stores)),
		TeamID: toTeamID(d.client.TeamID(config.TeamID.ValueString())),
	}
	for _, store := range stores {
		result.Stores = append(result.Stores, blobStoreListItemFromResponse(store))
	}

	tflog.Info(ctx, "read blob stores data source", map[string]any{
		"count":   len(result.Stores),
		"team_id": result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}
