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
	_ datasource.DataSource              = &blobProjectConnectionsDataSource{}
	_ datasource.DataSourceWithConfigure = &blobProjectConnectionsDataSource{}
)

func newBlobProjectConnectionsDataSource() datasource.DataSource {
	return &blobProjectConnectionsDataSource{}
}

type blobProjectConnectionsDataSource struct {
	client *client.Client
}

type BlobProjectConnectionsDataSourceModel struct {
	Connections []BlobProjectConnectionListItem `tfsdk:"connections"`
	StoreID     types.String                    `tfsdk:"store_id"`
	TeamID      types.String                    `tfsdk:"team_id"`
}

func (d *blobProjectConnectionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blob_project_connections"
}

func (d *blobProjectConnectionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *blobProjectConnectionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides the project connections for a Vercel Blob store.",
		Attributes: map[string]schema.Attribute{
			"store_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the Blob store whose project connections should be read.",
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team that owns the Blob store. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"connections": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The active project connections for the Blob store.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The ID of the Blob store project connection.",
						},
						"project_id": schema.StringAttribute{
							Computed:    true,
							Description: "The ID of the connected Vercel project.",
						},
						"project_name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the connected Vercel project.",
						},
						"project_framework": schema.StringAttribute{
							Computed:    true,
							Description: "The framework configured for the connected Vercel project, if any.",
						},
						"env_var_prefix": schema.StringAttribute{
							Computed:    true,
							Description: "The prefix used for the generated Blob environment variable names.",
						},
						"environments": schema.SetAttribute{
							Computed:    true,
							Description: "The environments in which the generated Blob environment variables exist.",
							ElementType: types.StringType,
						},
						"read_write_token_env_var_name": schema.StringAttribute{
							Computed:    true,
							Description: "The generated environment variable name that contains the Blob read/write token.",
						},
						"production_deployment_id": schema.StringAttribute{
							Computed:    true,
							Description: "The ID of the latest production deployment for the connected project, if any.",
						},
						"production_deployment_url": schema.StringAttribute{
							Computed:    true,
							Description: "The URL of the latest production deployment for the connected project, if any.",
						},
					},
				},
			},
		},
	}
}

func (d *blobProjectConnectionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config BlobProjectConnectionsDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	connections, err := d.client.ListBlobStoreConnections(ctx, config.StoreID.ValueString(), config.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Blob store project connections",
			fmt.Sprintf("Could not read Blob store project connections for store %s, unexpected error: %s", config.StoreID.ValueString(), err),
		)
		return
	}

	sortBlobConnections(connections)
	result := BlobProjectConnectionsDataSourceModel{
		Connections: make([]BlobProjectConnectionListItem, 0, len(connections)),
		StoreID:     config.StoreID,
		TeamID:      toTeamID(d.client.TeamID(config.TeamID.ValueString())),
	}
	for _, connection := range connections {
		item, diags := blobProjectConnectionListItemFromResponse(ctx, connection)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		result.Connections = append(result.Connections, item)
	}

	tflog.Info(ctx, "read blob project connections data source", map[string]any{
		"connection_count": len(result.Connections),
		"store_id":         result.StoreID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}
