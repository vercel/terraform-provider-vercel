package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v2/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &edgeConfigItemDataSource{}
	_ datasource.DataSourceWithConfigure = &edgeConfigItemDataSource{}
)

func newEdgeConfigItemDataSource() datasource.DataSource {
	return &edgeConfigItemDataSource{}
}

type edgeConfigItemDataSource struct {
	client *client.Client
}

func (d *edgeConfigItemDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_config_item"
}

func (d *edgeConfigItemDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Schema returns the schema information for an edgeConfigItem data source
func (r *edgeConfigItemDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides the value of an existing Edge Config Item.

An Edge Config is a global data store that enables experimentation with feature flags, A/B testing, critical redirects, and more.

An Edge Config Item is a value within an Edge Config.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the Edge Config that the item should exist under.",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Description: "The ID of the team the Edge Config should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
				Optional:    true,
				Computed:    true,
			},
			"key": schema.StringAttribute{
				Description: "The name of the key you want to retrieve within your Edge Config.",
				Required:    true,
			},
			"value": schema.StringAttribute{
				Description: "The value assigned to the key.",
				Computed:    true,
			},
		},
	}
}

type EdgeConfigItemDataSource struct {
	EdgeConfigID types.String `tfsdk:"id"`
	TeamID       types.String `tfsdk:"team_id"`
	Key          types.String `tfsdk:"key"`
	Value        types.String `tfsdk:"value"`
}

// Read will read the edgeConfigItem information by requesting it from the Vercel API, and will update terraform
// with this information.
// It is called by the provider whenever data source values should be read to update state.
func (d *edgeConfigItemDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config EdgeConfigItemDataSource
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetEdgeConfigItem(ctx, client.EdgeConfigItemRequest{
		EdgeConfigID: config.EdgeConfigID.ValueString(),
		TeamID:       config.TeamID.ValueString(),
		Key:          config.Key.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading EdgeConfigItem",
			fmt.Sprintf("Could not get Edge Config Item %s %s, unexpected error: %s",
				config.EdgeConfigID.ValueString(),
				config.Key.ValueString(),
				err,
			),
		)
		return
	}

	result := responseToEdgeConfigItem(out)
	tflog.Info(ctx, "read edge config item", map[string]interface{}{
		"edge_config_id": result.EdgeConfigID.ValueString(),
		"team_id":        result.TeamID.ValueString(),
		"key":            result.Key.ValueString(),
	})

	diags = resp.State.Set(ctx, EdgeConfigItemDataSource(result))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
