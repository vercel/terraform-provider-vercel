package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v2/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &customEnvironmentDataSource{}
)

func newCustomEnvironmentDataSource() datasource.DataSource {
	return &customEnvironmentDataSource{}
}

type customEnvironmentDataSource struct {
	client *client.Client
}

func (d *customEnvironmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_environment"
}

func (d *customEnvironmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Schema returns the schema information for an customEnvironment data source
func (r *customEnvironmentDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing CustomEnvironment resource.

An CustomEnvironment allows a ` + "`vercel_deployment` to be accessed through a different URL.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the environment.",
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The team ID to add the project to. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the existing Vercel Project.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the environment.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of what the environment is.",
				Computed:    true,
			},
			"branch_tracking": schema.SingleNestedAttribute{
				Description: "The branch tracking configuration for the environment. When enabled, each qualifying merge will generate a deployment.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"pattern": schema.StringAttribute{
						Description: "The pattern of the branch name to track.",
						Computed:    true,
					},
					"type": schema.StringAttribute{
						Description: "How a branch name should be matched against the pattern. Must be one of 'startsWith', 'endsWith' or 'equals'.",
						Computed:    true,
					},
				},
			},
		},
	}
}

// Read will read the customEnvironment information by requesting it from the Vercel API, and will update terraform
// with this information.
// It is called by the provider whenever data source values should be read to update state.
func (d *customEnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config CustomEnvironment
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := d.client.GetCustomEnvironment(ctx, client.GetCustomEnvironmentRequest{
		TeamID:    config.TeamID.ValueString(),
		ProjectID: config.ProjectID.ValueString(),
		Slug:      config.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading custom environment",
			fmt.Sprintf("Could not read custom environment %s %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ProjectID.ValueString(),
				config.Name.ValueString(),
				err,
			),
		)
		return
	}
	tflog.Trace(ctx, "read custom environment", map[string]any{
		"team_id":               config.TeamID.ValueString(),
		"project_id":            config.ProjectID.ValueString(),
		"custom_environment_id": res.ID,
	})

	diags = resp.State.Set(ctx, convertResponseToModel(res))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
