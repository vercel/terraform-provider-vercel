package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

var (
	_ datasource.DataSource              = &teamConfigDataSource{}
	_ datasource.DataSourceWithConfigure = &teamConfigDataSource{}
)

func newTeamConfigDataSource() datasource.DataSource {
	return &teamConfigDataSource{}
}

type teamConfigDataSource struct {
	client *client.Client
}

func (d *teamConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_config"
}

func (d *teamConfigDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *teamConfigDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the configuration of an existing Vercel Team.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the existing Vercel Team.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the team.",
				Computed:    true,
			},
			"invite_code": schema.StringAttribute{
				Computed:    true,
				Description: "A code that can be used to join this team. Only visible to Team owners.",
			},
			"slug": schema.StringAttribute{
				Description: "The slug of the team. Used in the URL of the team's dashboard.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "A description of the team.",
			},
			"sensitive_environment_variable_policy": schema.StringAttribute{
				Computed:    true,
				Description: "The policy for sensitive environment variables.",
			},
			"email_domain": schema.StringAttribute{
				Computed:    true,
				Description: "Hostname that'll be matched with emails on sign-up to automatically join the Team.",
			},
			"preview_deployment_suffix": schema.StringAttribute{
				Computed:    true,
				Description: "The hostname that is used as the preview deployment suffix.",
			},
			"remote_caching": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Configuration for Remote Caching.",
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Computed:    true,
						Description: "Indicates if Remote Caching is enabled.",
					},
				},
			},
			"enable_preview_feedback": schema.StringAttribute{
				Computed:    true,
				Description: "Preview feedback configuration.",
			},
			"enable_production_feedback": schema.StringAttribute{
				Computed:    true,
				Description: "Production feedback configuration.",
			},
			"hide_ip_addresses": schema.BoolAttribute{
				Computed:    true,
				Description: "Indicates if ip addresses should be accessible in o11y tooling.",
			},
			"hide_ip_addresses_in_log_drains": schema.BoolAttribute{
				Computed:    true,
				Description: "Indicates if ip addresses should be accessible in log drains.",
			},
			"on_demand_concurrent_builds": schema.BoolAttribute{
				Computed:    true,
				Description: "(Beta) Instantly scale build capacity to skip the queue, even if all build slots are in use. You can also choose a larger build machine; charges apply per minute if it exceeds your team's default.",
			},
			"saml": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"enforced": schema.BoolAttribute{
						Description: "Indicates if SAML is enforced for the team.",
						Computed:    true,
					},
					"roles": schema.MapNestedAttribute{
						Description: "Directory groups to role or access group mappings. For each directory group, either a role or access group id is specified.",
						Computed:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"role": schema.StringAttribute{
									Description: "The team level role the user is assigned. One of 'MEMBER', 'OWNER', 'VIEWER', 'DEVELOPER', 'BILLING' or 'CONTRIBUTOR'.",
									Computed:    true,
								},
								"access_group_id": schema.StringAttribute{
									Description: "The access group the assign is assigned to.",
									Computed:    true,
								},
							},
						},
					},
				},
				Computed:    true,
				Description: "Configuration for SAML authentication.",
			},
		},
	}
}

type TeamConfigData struct {
	ID                                 types.String `tfsdk:"id"`
	Name                               types.String `tfsdk:"name"`
	Slug                               types.String `tfsdk:"slug"`
	Description                        types.String `tfsdk:"description"`
	InviteCode                         types.String `tfsdk:"invite_code"`
	SensitiveEnvironmentVariablePolicy types.String `tfsdk:"sensitive_environment_variable_policy"`
	EmailDomain                        types.String `tfsdk:"email_domain"`
	PreviewDeploymentSuffix            types.String `tfsdk:"preview_deployment_suffix"`
	RemoteCaching                      types.Object `tfsdk:"remote_caching"`
	EnablePreviewFeedback              types.String `tfsdk:"enable_preview_feedback"`
	EnableProductionFeedback           types.String `tfsdk:"enable_production_feedback"`
	HideIPAddresses                    types.Bool   `tfsdk:"hide_ip_addresses"`
	HideIPAddressesInLogDrains         types.Bool   `tfsdk:"hide_ip_addresses_in_log_drains"`
	OnDemandConcurrentBuilds           types.Bool   `tfsdk:"on_demand_concurrent_builds"`
	Saml                               types.Object `tfsdk:"saml"`
}

func (d *teamConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config TeamConfigData
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID := config.ID.ValueString()
	team, err := d.client.GetTeam(ctx, teamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Team Config",
			fmt.Sprintf("Could not read Team Configuration with ID %s, unexpected error: %s", teamID, err),
		)
		return
	}

	out, diags := convertResponseToTeamConfig(ctx, team, types.MapNull(types.StringType))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, TeamConfigData{
		ID:                                 out.ID,
		Name:                               out.Name,
		Slug:                               out.Slug,
		Description:                        out.Description,
		InviteCode:                         out.InviteCode,
		SensitiveEnvironmentVariablePolicy: out.SensitiveEnvironmentVariablePolicy,
		EmailDomain:                        out.EmailDomain,
		PreviewDeploymentSuffix:            out.PreviewDeploymentSuffix,
		EnablePreviewFeedback:              out.EnablePreviewFeedback,
		EnableProductionFeedback:           out.EnableProductionFeedback,
		HideIPAddresses:                    out.HideIPAddresses,
		HideIPAddressesInLogDrains:         out.HideIPAddressesInLogDrains,
		OnDemandConcurrentBuilds:           out.OnDemandConcurrentBuilds,
		RemoteCaching:                      out.RemoteCaching,
		Saml:                               out.Saml,
	})
	resp.Diagnostics.Append(diags...)
}
