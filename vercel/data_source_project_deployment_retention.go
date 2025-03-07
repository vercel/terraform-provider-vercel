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

var (
	_ datasource.DataSource              = &projectDeploymentRetentionDataSource{}
	_ datasource.DataSourceWithConfigure = &projectDeploymentRetentionDataSource{}
)

func newProjectDeploymentRetentionDataSource() datasource.DataSource {
	return &projectDeploymentRetentionDataSource{}
}

type projectDeploymentRetentionDataSource struct {
	client *client.Client
}

func (r *projectDeploymentRetentionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_deployment_retention"
}

func (r *projectDeploymentRetentionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	r.client = client
}

// Schema returns the schema information for a project deployment retention datasource.
func (r *projectDeploymentRetentionDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Project Deployment Retention datasource.

A Project Deployment Retention datasource details information about Deployment Retention on a Vercel Project.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/security/deployment-retention).
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"expiration_preview": schema.StringAttribute{
				Computed:    true,
				Description: "The retention period for preview deployments.",
			},
			"expiration_production": schema.StringAttribute{
				Computed:    true,
				Description: "The retention period for production deployments.",
			},
			"expiration_canceled": schema.StringAttribute{
				Computed:    true,
				Description: "The retention period for canceled deployments.",
			},
			"expiration_errored": schema.StringAttribute{
				Computed:    true,
				Description: "The retention period for errored deployments.",
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the Project for the retention policy",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the Vercel team.",
			},
		},
	}
}

type ProjectDeploymentRetentionWithID struct {
	ExpirationPreview    types.String `tfsdk:"expiration_preview"`
	ExpirationProduction types.String `tfsdk:"expiration_production"`
	ExpirationCanceled   types.String `tfsdk:"expiration_canceled"`
	ExpirationErrored    types.String `tfsdk:"expiration_errored"`
	ProjectID            types.String `tfsdk:"project_id"`
	TeamID               types.String `tfsdk:"team_id"`
	ID                   types.String `tfsdk:"id"`
}

// Read will read an deployment retention of a Vercel project by requesting it from the Vercel API, and will update terraform
// with this information.
func (r *projectDeploymentRetentionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ProjectDeploymentRetentionWithID
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetDeploymentRetention(ctx, config.ProjectID.ValueString(), config.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project deployment retention",
			fmt.Sprintf("Could not get project deployment retention %s %s, unexpected error: %s",
				config.ProjectID.ValueString(),
				config.TeamID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToProjectDeploymentRetention(out, config.ProjectID, config.TeamID)
	tflog.Info(ctx, "read project deployment retention", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, ProjectDeploymentRetentionWithID{
		ExpirationPreview:    result.ExpirationPreview,
		ExpirationProduction: result.ExpirationProduction,
		ExpirationCanceled:   result.ExpirationCanceled,
		ExpirationErrored:    result.ExpirationErrored,
		ProjectID:            result.ProjectID,
		TeamID:               result.TeamID,
		ID:                   result.ProjectID,
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
