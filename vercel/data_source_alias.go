package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &aliasDataSource{}
)

func newAliasDataSource() datasource.DataSource {
	return &aliasDataSource{}
}

type aliasDataSource struct {
	client *client.Client
}

func (d *aliasDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alias"
}

func (d *aliasDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Schema returns the schema information for an alias data source
func (r *aliasDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Alias resource.

An Alias allows a ` + "`vercel_deployment` to be accessed through a different URL.",
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the Alias and Deployment exist under.",
			},
			"alias": schema.StringAttribute{
				Required:    true,
				Description: "The Alias or Alias ID to be retrieved.",
			},
			"deployment_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the Deployment the Alias is associated with.",
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Read will read the alias information by requesting it from the Vercel API, and will update terraform
// with this information.
// It is called by the provider whenever data source values should be read to update state.
func (d *aliasDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config Alias
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetAlias(ctx, config.Alias.ValueString(), config.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading alias",
			fmt.Sprintf("Could not read alias %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.Alias.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToAlias(out, config)
	tflog.Trace(ctx, "read alias", map[string]interface{}{
		"team_id": result.TeamID.ValueString(),
		"alias":   result.Alias.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
