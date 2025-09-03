package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

var (
	_ datasource.DataSource              = &projectMembersDataSource{}
	_ datasource.DataSourceWithConfigure = &projectMembersDataSource{}
)

func newProjectMembersDataSource() datasource.DataSource {
	return &projectMembersDataSource{}
}

type projectMembersDataSource struct {
	client *client.Client
}

func (d *projectMembersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_members"
}

func (d *projectMembersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *projectMembersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves members and their roles for a Vercel Project.",
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The team ID to which the project belongs. Required when accessing a team project if a default team has not been set in the provider.",
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the Vercel Project.",
			},
			"members": schema.SetNestedAttribute{
				Computed:    true,
				Description: "The set of members in this project.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"user_id": schema.StringAttribute{
							Computed:    true,
							Description: "The ID of the user.",
						},
						"email": schema.StringAttribute{
							Computed:    true,
							Description: "The email of the user.",
						},
						"username": schema.StringAttribute{
							Computed:    true,
							Description: "The username of the user.",
						},
						"role": schema.StringAttribute{
							Computed:    true,
							Description: "The role of the user in the project. One of 'ADMIN', 'PROJECT_DEVELOPER', or 'PROJECT_VIEWER'.",
						},
					},
				},
			},
		},
	}
}

type ProjectMembersDataSourceModel struct {
	TeamID    types.String `tfsdk:"team_id"`
	ProjectID types.String `tfsdk:"project_id"`
	Members   types.Set    `tfsdk:"members"`
}

func (d *projectMembersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ProjectMembersDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	members, err := d.client.ListProjectMembers(ctx, client.GetProjectMembersRequest{
		TeamID:    config.TeamID.ValueString(),
		ProjectID: config.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Project Members",
			fmt.Sprintf("Could not read Project Members, unexpected error: %s", err),
		)
		return
	}

	// Convert API response to model
	var memberItems []attr.Value
	for _, member := range members {
		memberItems = append(memberItems, types.ObjectValueMust(memberAttrType.AttrTypes, map[string]attr.Value{
			"user_id":  types.StringValue(member.UserID),
			"email":    types.StringValue(member.Email),
			"username": types.StringValue(member.Username),
			"role":     types.StringValue(member.Role),
		}))
	}

	config.Members = types.SetValueMust(memberAttrType, memberItems)
	config.TeamID = types.StringValue(d.client.TeamID(config.TeamID.ValueString()))
	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}
