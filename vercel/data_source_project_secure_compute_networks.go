package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

var (
	_ datasource.DataSource              = &projectSecureComputeNetworksDataSource{}
	_ datasource.DataSourceWithConfigure = &projectSecureComputeNetworksDataSource{}
)

func newProjectSecureComputeNetworksDataSource() datasource.DataSource {
	return &projectSecureComputeNetworksDataSource{}
}

type projectSecureComputeNetworksDataSource struct {
	client *client.Client
}

func (r *projectSecureComputeNetworksDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_secure_compute_networks"
}

func (r *projectSecureComputeNetworksDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Schema returns the schema information for a microfrontendGroup data source.
func (r *projectSecureComputeNetworksDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: ` `,
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The ID of the Project",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Description: "The team ID. Required when configuring a team data source if a default team has not been set in the provider.",
				Optional:    true,
				Computed:    true,
			},
			"secure_compute_networks": schema.SetNestedAttribute{
				Description: "A set of Secure Compute Networks that the project should be configured with.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"environment": schema.StringAttribute{
							Description: "The environment being configured. Should be one of 'production', 'preview', or the ID of a Custom Environment",
							Computed:    true,
						},
						"network_id": schema.StringAttribute{
							Description: "The ID of the Secure Compute Network to configure for this environment",
							Computed:    true,
						},
						"passive": schema.BoolAttribute{
							Description: "Whether the Secure Compute Network should be configured as a passive network, meaning it is used for passive failover.",
							Computed:    true,
						},
						"builds_enabled": schema.BoolAttribute{
							Description: "Whether the projects build container should be included in the Secure Compute Network.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *projectSecureComputeNetworksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ProjectSecureComputeNetworks
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetProject(ctx, config.ProjectID.ValueString(), config.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading projject secure compute networks",
			fmt.Sprintf("Could not get project secure compute networks %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	diags = resp.State.Set(ctx, convertResponseToProjectSecureComputeNetworks(out))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
