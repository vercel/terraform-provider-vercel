package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type dataSourceAliasType struct{}

// GetSchema returns the schema information for an alias data source
func (r dataSourceAliasType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: `
Provides information about an existing alias within Vercel.

An alias allows a deployment to be accessed through a different URL.
        `,
		Attributes: map[string]tfsdk.Attribute{
			"team_id": {
				Optional:      true,
				Type:          types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{tfsdk.RequiresReplace()},
				Description:   "The team ID the alias exists beneath.",
			},
			"alias": {
				Required:    true,
				Type:        types.StringType,
				Description: "The alias or alias ID to be retrieved.",
			},
			"deployment_id": {
				Computed:    true,
				Type:        types.StringType,
				Description: "The deployment ID.",
			},
			"uid": {
				Computed:    true,
				Type:        types.StringType,
				Description: "The unique identifier of the alias.",
			},
		},
	}, nil
}

// NewDataSource instantiates a new DataSource of this DataSourceType.
func (r dataSourceAliasType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	return dataSourceAlias{
		p: *(p.(*provider)),
	}, nil
}

type dataSourceAlias struct {
	p provider
}

// Read will read the alias information by requesting it from the Vercel API, and will update terraform
// with this information.
// It is called by the provider whenever data source values should be read to update state.
func (r dataSourceAlias) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
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
	tflog.Trace(ctx, "read project", map[string]interface{}{
		"team_id": result.TeamID.Value,
		"alias":   result.Alias.Value,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
