package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

var (
	_ datasource.DataSource              = &dsyncGroupsDataSource{}
	_ datasource.DataSourceWithConfigure = &dsyncGroupsDataSource{}
)

func newDsyncGroupsDataSource() datasource.DataSource {
	return &dsyncGroupsDataSource{}
}

type dsyncGroupsDataSource struct {
	client *client.Client
}

func (d *dsyncGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dsync_groups"
}

func (d *dsyncGroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *dsyncGroupsDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about DSync groups for a team.",
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Description: "The ID of the team to retrieve DSync groups for.",
				Required:    true,
			},
			"groups": schema.MapNestedAttribute{
				Description: "A map of DSync groups for the team.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the group on Vercel.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the group on the Identity Provider.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

type DsyncGroup struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type DsyncGroups struct {
	TeamID types.String          `tfsdk:"team_id"`
	Groups map[string]DsyncGroup `tfsdk:"groups"`
}

func responseToDsyncGroups(out client.DsyncGroups) DsyncGroups {
	return DsyncGroups{
		Groups: make(map[string]DsyncGroup, len(out.Groups)),
	}
}

func (d *dsyncGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DsyncGroups
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetDsyncGroups(ctx, config.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading DSync Groups",
			fmt.Sprintf("Could not get DSync Groups for team %s, unexpected error: %s",
				config.TeamID.ValueString(),
				err,
			),
		)
		return
	}

	result := responseToDsyncGroups(out)
	tflog.Info(ctx, "read dsync groups", map[string]any{
		"team_id": result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
