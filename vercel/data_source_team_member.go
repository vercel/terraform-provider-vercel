package vercel

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &teamMemberDataSource{}
	_ datasource.DataSourceWithConfigure = &teamMemberDataSource{}
)

func newTeamMemberDataSource() datasource.DataSource {
	return &teamMemberDataSource{}
}

type teamMemberDataSource struct {
	client *client.Client
}

func (d *teamMemberDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_member"
}

func (d *teamMemberDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (d *teamMemberDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provider a datasource for managing a team member.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"team_id": schema.StringAttribute{
				Description: "The ID of the existing Vercel Team.",
				Required:    true,
			},
			"user_id": schema.StringAttribute{
				Description: "The ID of the existing Vercel Team Member.",
				Required:    true,
			},
			"email": schema.StringAttribute{
				Description: "The email address of the existing Vercel Team Member.",
				Computed:    true,
			},
			"role": schema.StringAttribute{
				Description: "The role that the user should have in the project. One of 'MEMBER', 'OWNER', 'VIEWER', 'DEVELOPER', 'BILLING' or 'CONTRIBUTOR'. Depending on your Team's plan, some of these roles may be unavailable.",
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("MEMBER", "OWNER", "VIEWER", "DEVELOPER", "BILLING", "CONTRIBUTOR"),
				},
			},
			"projects": schema.SetNestedAttribute{
				Description: "If access groups are enabled on the team, and the user is a CONTRIBUTOR, `projects`, `access_groups` or both must be specified. A set of projects that the user should be granted access to, along with their role in each project.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role": schema.StringAttribute{
							Description: "The role that the user should have in the project.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("ADMIN", "PROJECT_VIEWER", "PROJECT_DEVELOPER"),
							},
						},
						"project_id": schema.StringAttribute{
							Description: "The ID of the project that the user should be granted access to.",
							Required:    true,
						},
					},
				},
			},
			"access_groups": schema.SetAttribute{
				Description: "If access groups are enabled on the team, and the user is a CONTRIBUTOR, `projects`, `access_groups` or both must be specified. A set of access groups IDs that the user should be granted access to.",
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
		},
	}
}

type TeamMemberWithID struct {
	UserID       types.String `tfsdk:"user_id"`
	Email        types.String `tfsdk:"email"`
	TeamID       types.String `tfsdk:"team_id"`
	Role         types.String `tfsdk:"role"`
	Projects     types.Set    `tfsdk:"projects"`
	AccessGroups types.Set    `tfsdk:"access_groups"`
	ID           types.String `tfsdk:"id"`
}

func (d *teamMemberDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config TeamMemberWithID
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var response client.TeamMember
	getRetry := Retry{
		Base:     200 * time.Millisecond,
		Attempts: 7,
	}
	err := getRetry.Do(func(attempt int) (shouldRetry bool, err error) {
		response, err = d.client.GetTeamMember(ctx, client.GetTeamMemberRequest{
			TeamID: config.TeamID.ValueString(),
			UserID: config.UserID.ValueString(),
		})
		if client.NotFound(err) {
			return true, err
		}
		if err != nil {
			return true, fmt.Errorf("unexpected error: %w", err)
		}
		return false, err
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Team Member",
			"Could not read Team Member, unexpected error: "+err.Error(),
		)
	}
	teamMember := convertResponseToTeamMember(response, TeamMember{
		UserID:       config.UserID,
		Email:        config.Email,
		TeamID:       config.TeamID,
		Role:         config.Role,
		Projects:     config.Projects,
		AccessGroups: config.AccessGroups,
	})
	diags = resp.State.Set(ctx, TeamMemberWithID{
		UserID:       teamMember.UserID,
		TeamID:       teamMember.TeamID,
		Email:        teamMember.Email,
		Role:         teamMember.Role,
		Projects:     teamMember.Projects,
		AccessGroups: teamMember.AccessGroups,
		ID:           types.StringValue(fmt.Sprintf("%s/%s", config.TeamID.ValueString(), config.UserID.ValueString())),
	})
	resp.Diagnostics.Append(diags...)
}
