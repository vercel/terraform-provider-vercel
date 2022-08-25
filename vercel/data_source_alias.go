package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type dataSourceAliasType struct{}

// GetSchema returns the schema information for an alias data source
func (r dataSourceAliasType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: `
Provides information about an existing Alias resource.

An Alias allows a ` + "`vercel_deployment` to be accessed through a different URL.",
		Attributes: map[string]tfsdk.Attribute{
			"team_id": {
				Optional:    true,
				Type:        types.StringType,
				Description: "The ID of the team the Alias and Deployment exist under.",
			},
			"alias": {
				Required:    true,
				Type:        types.StringType,
				Description: "The Alias or Alias ID to be retrieved.",
			},
			"deployment_id": {
				Computed:    true,
				Type:        types.StringType,
				Description: "The ID of the Deployment the Alias is associated with.",
			},
			"id": {
				Computed: true,
				Type:     types.StringType,
			},
		},
	}, nil
}

// NewDataSource instantiates a new DataSource of this DataSourceType.
func (r dataSourceAliasType) NewDataSource(ctx context.Context, p provider.Provider) (datasource.DataSource, diag.Diagnostics) {
	return dataSourceAlias{
		p: *(p.(*vercelProvider)),
	}, nil
}

type dataSourceAlias struct {
	p vercelProvider
}

// Read will read the alias information by requesting it from the Vercel API, and will update terraform
// with this information.
// It is called by the provider whenever data source values should be read to update state.
func (r dataSourceAlias) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config Alias
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.p.client.GetAlias(ctx, config.Alias.Value, config.TeamID.Value)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading alias",
			fmt.Sprintf("Could not read alias %s %s, unexpected error: %s",
				config.TeamID.Value,
				config.Alias.Value,
				err,
			),
		)
		return
	}

	result := convertResponseToAlias(out, config)
	tflog.Info(ctx, "read alias", map[string]interface{}{
		"team_id": result.TeamID.Value,
		"alias":   result.Alias.Value,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
