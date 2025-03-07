package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v2/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &edgeConfigTokenDataSource{}
	_ datasource.DataSourceWithConfigure = &edgeConfigTokenDataSource{}
)

func newEdgeConfigTokenDataSource() datasource.DataSource {
	return &edgeConfigTokenDataSource{}
}

type edgeConfigTokenDataSource struct {
	client *client.Client
}

func (d *edgeConfigTokenDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_config_token"
}

func (d *edgeConfigTokenDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

// Schema returns the schema information for an edgeConfigToken data source
func (r *edgeConfigTokenDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Edge Config Token.

An Edge Config is a global data store that enables experimentation with feature flags, A/B testing, critical redirects, and more.

An Edge Config token is used to authenticate against an Edge Config's endpoint.
`,
		Attributes: map[string]schema.Attribute{
			"label": schema.StringAttribute{
				Description: "The label of the Edge Config Token.",
				Computed:    true,
			},
			"edge_config_id": schema.StringAttribute{
				Description: "The label of the Edge Config Token.",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the Edge Config should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
			"token": schema.StringAttribute{
				Description: "A read access token used for authenticating against the Edge Config's endpoint for high volume, low-latency requests.",
				Required:    true,
			},
			"connection_string": schema.StringAttribute{
				Description: "A connection string is a URL that connects a project to an Edge Config. The variable can be called anything, but our Edge Config client SDK will search for process.env.EDGE_CONFIG by default.",
				Computed:    true,
			},
		},
	}
}

// Read will read the edgeConfigToken information by requesting it from the Vercel API, and will update terraform
// with this information.
// It is called by the provider whenever data source values should be read to update state.
func (d *edgeConfigTokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config EdgeConfigToken
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetEdgeConfigToken(ctx, client.EdgeConfigTokenRequest{
		Token:        config.Token.ValueString(),
		TeamID:       config.TeamID.ValueString(),
		EdgeConfigID: config.EdgeConfigID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading EdgeConfig Token",
			fmt.Sprintf("Could not get Edge Config Token %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := responseToEdgeConfigToken(out)
	tflog.Info(ctx, "read edge config token", map[string]any{
		"team_id":        result.TeamID.ValueString(),
		"edge_config_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
