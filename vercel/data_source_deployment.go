package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &deploymentDataSource{}
	_ datasource.DataSourceWithConfigure = &deploymentDataSource{}
)

func newDeploymentDataSource() datasource.DataSource {
	return &deploymentDataSource{}
}

type deploymentDataSource struct {
	client *client.Client
}

func (d *deploymentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

func (d *deploymentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Schema returns the schema information for an deployment data source
func (r *deploymentDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Deployment.

A Deployment is the result of building your Project and making it available through a live URL.
`,
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Description: "The Team ID to the Deployment belong to. Required when reading a team resource if a default team has not been set in the provider.",
				Optional:    true,
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The ID or URL of the Deployment to read.",
			},
			"domains": schema.ListAttribute{
				Description: "A list of all the domains (default domains, staging domains and production domains) that were assigned upon deployment creation.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"project_id": schema.StringAttribute{
				Description: "The project ID to add the deployment to.",
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "A unique URL that is automatically generated for a deployment.",
				Computed:    true,
			},
			"production": schema.BoolAttribute{
				Description: "true if the deployment is a production deployment, meaning production aliases will be assigned.",
				Computed:    true,
			},
			"ref": schema.StringAttribute{
				Description: "The branch or commit hash that has been deployed. Note this will only work if the project is configured to use a Git repository.",
				Computed:    true,
			},
			"meta": schema.MapAttribute{
				Description: "Arbitrary key/value metadata associated with the deployment.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"custom_environment_id": schema.StringAttribute{
				Description: "The ID of the Custom Environment that the deployment was deployed to, if any.",
				Computed:    true,
			},
		},
	}
}

type DeploymentDataSource struct {
	Domains             types.List   `tfsdk:"domains"`
	ID                  types.String `tfsdk:"id"`
	Production          types.Bool   `tfsdk:"production"`
	ProjectID           types.String `tfsdk:"project_id"`
	TeamID              types.String `tfsdk:"team_id"`
	URL                 types.String `tfsdk:"url"`
	Ref                 types.String `tfsdk:"ref"`
	Meta                types.Map    `tfsdk:"meta"`
	CustomEnvironmentID types.String `tfsdk:"custom_environment_id"`
}

func convertResponseToDeploymentDataSource(in client.DeploymentResponse) DeploymentDataSource {
	ref := types.StringNull()
	if in.GitSource.Ref != "" {
		ref = types.StringValue(in.GitSource.Ref)
	}

	var domains []attr.Value
	for _, a := range in.Aliases {
		domains = append(domains, types.StringValue(a))
	}

	customEnvironmentID := types.StringNull()
	if in.CustomEnvironment != nil && in.CustomEnvironment.ID != "" {
		customEnvironmentID = types.StringValue(in.CustomEnvironment.ID)
	}

	metaAttrs := map[string]attr.Value{}
	for k, v := range in.Meta {
		metaAttrs[k] = types.StringValue(v)
	}

	return DeploymentDataSource{
		Domains:             types.ListValueMust(types.StringType, domains),
		Production:          types.BoolValue(in.Target != nil && *in.Target == "production"),
		TeamID:              toTeamID(in.TeamID),
		ProjectID:           types.StringValue(in.ProjectID),
		ID:                  types.StringValue(in.ID),
		URL:                 types.StringValue(in.URL),
		Ref:                 ref,
		Meta:                types.MapValueMust(types.StringType, metaAttrs),
		CustomEnvironmentID: customEnvironmentID,
	}
}

// Read will read the deployment information by requesting it from the Vercel API, and will update terraform
// with this information.
// It is called by the provider whenever data source values should be read to update state.
func (d *deploymentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DeploymentDataSource
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetDeployment(ctx, config.ID.ValueString(), config.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading deployment",
			fmt.Sprintf("Could not get deployment %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToDeploymentDataSource(out)
	tflog.Info(ctx, "read deployment", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
