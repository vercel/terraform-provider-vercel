package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var (
	_ datasource.DataSource              = &projectDelegatedProtectionDataSource{}
	_ datasource.DataSourceWithConfigure = &projectDelegatedProtectionDataSource{}
)

func newProjectDelegatedProtectionDataSource() datasource.DataSource {
	return &projectDelegatedProtectionDataSource{}
}

type projectDelegatedProtectionDataSource struct {
	client *client.Client
}

type ProjectDelegatedProtectionDataSource struct {
	ID             types.String `tfsdk:"id"`
	ProjectID      types.String `tfsdk:"project_id"`
	TeamID         types.String `tfsdk:"team_id"`
	ClientID       types.String `tfsdk:"client_id"`
	CookieName     types.String `tfsdk:"cookie_name"`
	DeploymentType types.String `tfsdk:"deployment_type"`
	Issuer         types.String `tfsdk:"issuer"`
}

func (d *projectDelegatedProtectionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_delegated_protection"
}

func (d *projectDelegatedProtectionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *projectDelegatedProtectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a Project Delegated Protection data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for this data source.",
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the Project to read delegated protection for.",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the Project exists under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"client_id": schema.StringAttribute{
				Computed:    true,
				Description: "The OAuth client ID used for delegated protection.",
			},
			"cookie_name": schema.StringAttribute{
				Computed:    true,
				Description: "The cookie name used for delegated protection.",
			},
			"deployment_type": schema.StringAttribute{
				Computed:    true,
				Description: "The deployment environment protected by delegated protection.",
			},
			"issuer": schema.StringAttribute{
				Computed:    true,
				Description: "The issuer URL of the OIDC provider used for delegated protection.",
			},
		},
	}
}

func projectDelegatedProtectionDataSourceFromResponse(response client.DelegatedProtection) ProjectDelegatedProtectionDataSource {
	cookieName := types.StringNull()
	if response.CookieName != nil {
		cookieName = types.StringValue(*response.CookieName)
	}

	return ProjectDelegatedProtectionDataSource{
		ID:             types.StringValue(response.ProjectID),
		ProjectID:      types.StringValue(response.ProjectID),
		TeamID:         toTeamID(response.TeamID),
		ClientID:       types.StringValue(response.ClientID),
		CookieName:     cookieName,
		DeploymentType: fromApiDeploymentProtectionType(response.DeploymentType),
		Issuer:         types.StringValue(response.Issuer),
	}
}

func (d *projectDelegatedProtectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ProjectDelegatedProtectionDataSource
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetProjectDelegatedProtection(ctx, config.ProjectID.ValueString(), config.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project delegated protection",
			fmt.Sprintf("Could not get project delegated protection %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	result := projectDelegatedProtectionDataSourceFromResponse(out)
	tflog.Info(ctx, "read project delegated protection", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}
