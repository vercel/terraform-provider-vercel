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
	_ datasource.DataSource              = &bulkRedirectsDataSource{}
	_ datasource.DataSourceWithConfigure = &bulkRedirectsDataSource{}
)

func newBulkRedirectsDataSource() datasource.DataSource {
	return &bulkRedirectsDataSource{}
}

type bulkRedirectsDataSource struct {
	client *client.Client
}

type bulkRedirectsDataSourceModel struct {
	ProjectID types.String `tfsdk:"project_id"`
	TeamID    types.String `tfsdk:"team_id"`
	VersionID types.String `tfsdk:"version_id"`
	Redirects types.Set    `tfsdk:"redirects"`
}

func (d *bulkRedirectsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bulk_redirects"
}

func (d *bulkRedirectsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *bulkRedirectsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Bulk Redirects data source.

If version_id is omitted, the data source reads the live production bulk redirects for the project.
`,
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The ID of the Vercel project.",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the Vercel team.",
			},
			"version_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The redirects version to read. If omitted, the live production version is used.",
			},
			"redirects": schema.SetNestedAttribute{
				Computed:    true,
				Description: "The redirects for the selected version.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"source": schema.StringAttribute{
							Computed:    true,
							Description: "The source pathname to match.",
						},
						"destination": schema.StringAttribute{
							Computed:    true,
							Description: "The destination pathname or URL to redirect to.",
						},
						"status_code": schema.Int64Attribute{
							Computed:    true,
							Description: "The HTTP status code for the redirect.",
						},
						"case_sensitive": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether the source match is case-sensitive.",
						},
						"query": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether query parameters are considered when matching the redirect.",
						},
					},
				},
			},
		},
	}
}

func (d *bulkRedirectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config bulkRedirectsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var (
		out client.BulkRedirects
		err error
	)

	if !config.VersionID.IsNull() && !config.VersionID.IsUnknown() && config.VersionID.ValueString() != "" {
		out, err = d.client.GetBulkRedirects(ctx, client.GetBulkRedirectsRequest{
			ProjectID: config.ProjectID.ValueString(),
			TeamID:    config.TeamID.ValueString(),
			VersionID: config.VersionID.ValueString(),
		})
	} else {
		var live bool
		out, live, err = readLiveBulkRedirects(ctx, d.client, config.ProjectID.ValueString(), config.TeamID.ValueString())
		if err == nil && !live {
			resp.Diagnostics.AddError(
				"Error reading bulk redirects",
				fmt.Sprintf("No live bulk redirects version exists for project %s", config.ProjectID.ValueString()),
			)
			return
		}
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading bulk redirects",
			fmt.Sprintf("Could not get bulk redirects %s %s, unexpected error: %s", config.TeamID.ValueString(), config.ProjectID.ValueString(), err),
		)
		return
	}

	config.TeamID = toTeamID(out.TeamID)
	config.Redirects = flattenBulkRedirects(out.Redirects)
	if out.Version != nil {
		config.VersionID = types.StringValue(out.Version.ID)
	}

	tflog.Info(ctx, "read bulk redirects data source", map[string]any{
		"project_id": config.ProjectID.ValueString(),
		"team_id":    config.TeamID.ValueString(),
		"version_id": config.VersionID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
