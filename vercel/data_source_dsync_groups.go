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
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the Dsync Groups are associated to. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"list": schema.ListNestedAttribute{
				Description: "A list of DSync groups for the team.",
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
			"map": schema.MapAttribute{
				Description: "A map of Identity Provider group names to their Vercel IDs. This can be used to look up the ID of a group by its name using the [lookup](https://developer.hashicorp.com/terraform/language/functions/lookup) function.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

type DsyncGroup struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type DsyncGroups struct {
	TeamID types.String            `tfsdk:"team_id"`
	List   []DsyncGroup            `tfsdk:"list"`
	Map    map[string]types.String `tfsdk:"map"`
}

func (d *dsyncGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DsyncGroups
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetDsyncGroups(ctx, config.TeamID.ValueString())
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

	var groupsList = make([]DsyncGroup, 0, len(out.Groups))
	var groupsMap = make(map[string]types.String, len(out.Groups))
	for _, g := range out.Groups {
		groupsList = append(groupsList, DsyncGroup{
			ID:   types.StringValue(g.ID),
			Name: types.StringValue(g.Name),
		})
		groupsMap[g.Name] = types.StringValue(g.ID)
	}

	result := DsyncGroups{
		TeamID: types.StringValue(out.TeamID),
		List:   groupsList,
		Map:    groupsMap,
	}

	tflog.Info(ctx, "read dsync groups", map[string]any{
		"team_id": result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, DsyncGroups(result))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
