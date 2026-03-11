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
	_ datasource.DataSource              = &blobStoreSecretsDataSource{}
	_ datasource.DataSourceWithConfigure = &blobStoreSecretsDataSource{}
)

func newBlobStoreSecretsDataSource() datasource.DataSource {
	return &blobStoreSecretsDataSource{}
}

type blobStoreSecretsDataSource struct {
	client *client.Client
}

type BlobStoreSecretsDataSourceModel struct {
	ReadWriteToken types.String `tfsdk:"read_write_token"`
	StoreID        types.String `tfsdk:"store_id"`
	TeamID         types.String `tfsdk:"team_id"`
}

func (d *blobStoreSecretsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blob_store_secrets"
}

func (d *blobStoreSecretsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *blobStoreSecretsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides the default read/write token for a Vercel Blob store.",
		Attributes: map[string]schema.Attribute{
			"store_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the Blob store whose secrets should be read.",
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team that owns the Blob store. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"read_write_token": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The default Blob read/write token for the store.",
			},
		},
	}
}

func (d *blobStoreSecretsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config BlobStoreSecretsDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secrets, err := d.client.GetBlobStoreSecrets(ctx, config.StoreID.ValueString(), config.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Blob store secrets",
			fmt.Sprintf("Could not read Blob store secrets for store %s, unexpected error: %s", config.StoreID.ValueString(), err),
		)
		return
	}

	result := BlobStoreSecretsDataSourceModel{
		ReadWriteToken: types.StringValue(secrets.ReadWriteToken),
		StoreID:        config.StoreID,
		TeamID:         toTeamID(d.client.TeamID(config.TeamID.ValueString())),
	}

	tflog.Info(ctx, "read blob store secrets data source", map[string]any{
		"store_id": result.StoreID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}
