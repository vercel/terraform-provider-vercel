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
	_ datasource.DataSource              = &blobObjectDataSource{}
	_ datasource.DataSourceWithConfigure = &blobObjectDataSource{}
)

func newBlobObjectDataSource() datasource.DataSource {
	return &blobObjectDataSource{}
}

type blobObjectDataSource struct {
	client *client.Client
}

func (d *blobObjectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blob_object"
}

func (d *blobObjectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *blobObjectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides metadata about an existing object in a Vercel Blob store.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for this Blob object. Format: `store_id/pathname`.",
			},
			"store_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the Blob store that contains the object.",
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team that owns the Blob store. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"pathname": schema.StringAttribute{
				Required:    true,
				Description: "The pathname of the Blob object within the store.",
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "The canonical URL for the Blob object.",
			},
			"download_url": schema.StringAttribute{
				Computed:    true,
				Description: "The Blob object URL with download semantics enabled.",
			},
			"size": schema.Int64Attribute{
				Computed:    true,
				Description: "The size of the Blob object in bytes.",
			},
			"uploaded_at": schema.StringAttribute{
				Computed:    true,
				Description: "The timestamp at which the Blob object was uploaded.",
			},
			"content_type": schema.StringAttribute{
				Computed:    true,
				Description: "The content type stored on the Blob object.",
			},
			"content_disposition": schema.StringAttribute{
				Computed:    true,
				Description: "The content disposition returned for the Blob object.",
			},
			"cache_control": schema.StringAttribute{
				Computed:    true,
				Description: "The full Cache-Control header stored on the Blob object.",
			},
			"cache_control_max_age": schema.Int64Attribute{
				Computed:    true,
				Description: "The cache max-age, in seconds, stored on the Blob object.",
			},
			"etag": schema.StringAttribute{
				Computed:    true,
				Description: "The current ETag for the Blob object.",
			},
		},
	}
}

func (d *blobObjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config BlobObjectDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	object, err := d.client.GetBlobObject(ctx, client.GetBlobObjectRequest{
		Pathname: config.Pathname.ValueString(),
		StoreID:  config.StoreID.ValueString(),
		TeamID:   config.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Blob object",
			fmt.Sprintf("Could not read Blob object %s, unexpected error: %s", blobObjectID(config.StoreID.ValueString(), config.Pathname.ValueString()), err),
		)
		return
	}

	result := blobObjectDataSourceModelFromResponse(config.StoreID.ValueString(), d.client.TeamID(config.TeamID.ValueString()), object)
	tflog.Info(ctx, "read blob object data source", map[string]any{
		"blob_object_id": result.ID.ValueString(),
		"store_id":       result.StoreID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}
