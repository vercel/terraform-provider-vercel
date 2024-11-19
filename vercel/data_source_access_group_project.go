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
	_ datasource.DataSource              = &accessGroupProjectDataSource{}
	_ datasource.DataSourceWithConfigure = &accessGroupProjectDataSource{}
)

func newAccessGroupProjectDataSource() datasource.DataSource {
	return &accessGroupProjectDataSource{}
}

type accessGroupProjectDataSource struct {
	client *client.Client
}

func (d *accessGroupProjectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_group_project"
}

func (d *accessGroupProjectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *accessGroupProjectDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Access Group Project Assignment.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/accounts/team-members-and-roles/access-groups).
`,
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the Access Group Project should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"access_group_id": schema.StringAttribute{
				Required:    true,
				Description: "The Access Group ID.",
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The Project ID.",
			},
			"role": schema.StringAttribute{
				Computed:    true,
				Description: "The Access Group Project Role.",
			},
		},
	}
}

func (d *accessGroupProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state AccessGroupProject
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetAccessGroupProject(ctx, client.GetAccessGroupProjectRequest{
		TeamID:        state.TeamID.ValueString(),
		AccessGroupID: state.AccessGroupID.ValueString(),
		ProjectID:     state.ProjectID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Access Group Project",
			fmt.Sprintf("Could not get Access Group Project %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.AccessGroupID.ValueString(),
				state.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	result := AccessGroupProject{
		TeamID:        types.StringValue(out.TeamID),
		AccessGroupID: types.StringValue(out.AccessGroupID),
		ProjectID:     types.StringValue(out.ProjectID),
		Role:          types.StringValue(out.Role),
	}
	tflog.Info(ctx, "read Access Group Project", map[string]interface{}{
		"team_id":         result.TeamID.ValueString(),
		"access_group_id": result.AccessGroupID.ValueString(),
		"project_id":      result.ProjectID.ValueString(),
		"role":            result.Role.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
