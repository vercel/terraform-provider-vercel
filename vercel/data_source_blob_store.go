package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var (
	_ datasource.DataSource              = &blobStoreDataSource{}
	_ datasource.DataSourceWithConfigure = &blobStoreDataSource{}
)

func newBlobStoreDataSource() datasource.DataSource {
	return &blobStoreDataSource{}
}

type blobStoreDataSource struct {
	client *client.Client
}

func (d *blobStoreDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blob_store"
}

func (d *blobStoreDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *blobStoreDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about an existing Vercel Blob store.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the Blob store.",
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team that owns the Blob store. Required when configuring a team resource if a default team has not been set in the provider.",
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
	}
}

func (d *blobStoreDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config BlobStoreModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	store, err := d.client.GetBlobStore(ctx, config.ID.ValueString(), config.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Blob store",
			fmt.Sprintf("Could not read Blob store %s %s, unexpected error: %s", config.TeamID.ValueString(), config.ID.ValueString(), err),
		)
		return
	}

	result := blobStoreModelFromResponse(store)
	tflog.Info(ctx, "read blob store data source", map[string]any{
		"blob_store_id": result.ID.ValueString(),
		"team_id":       result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}
