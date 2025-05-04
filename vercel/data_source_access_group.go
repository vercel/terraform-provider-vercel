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

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &accessGroupDataSource{}
	_ datasource.DataSourceWithConfigure = &accessGroupDataSource{}
)

func newAccessGroupDataSource() datasource.DataSource {
	return &accessGroupDataSource{}
}

type accessGroupDataSource struct {
	client *client.Client
}

func (d *accessGroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_group"
}

func (d *accessGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *accessGroupDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Access Group.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/accounts/team-members-and-roles/access-groups).
`,
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the Access Group should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The Access Group ID to be retrieved.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the Access Group.",
			},
		},
	}
}

func (d *accessGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state AccessGroup
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetAccessGroup(ctx, client.GetAccessGroupRequest{
		AccessGroupID: state.ID.ValueString(),
		TeamID:        state.TeamID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Access Group",
			fmt.Sprintf("Could not get Access Group %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := AccessGroup{
		ID:     types.StringValue(out.ID),
		TeamID: types.StringValue(out.TeamID),
		Name:   types.StringValue(out.Name),
	}
	tflog.Info(ctx, "read Access Group", map[string]any{
		"team_id":         result.TeamID.ValueString(),
		"access_group_id": result.ID.ValueString(),
		"name":            result.Name.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
