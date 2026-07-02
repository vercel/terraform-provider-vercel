package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ datasource.DataSource              = &vcrRepositoryDataSource{}
	_ datasource.DataSourceWithConfigure = &vcrRepositoryDataSource{}
)

func newVCRRepositoryDataSource() datasource.DataSource {
	return &vcrRepositoryDataSource{}
}

type vcrRepositoryDataSource struct {
	client *client.Client
}

func (d *vcrRepositoryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vcr_repository"
}

func (d *vcrRepositoryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *vcrRepositoryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Vercel Container Registry (VCR) Repository.

A VCR Repository belongs to a Vercel Project and stores container images that can be
used by Vercel Functions and Vercel Sandbox.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the VCR Repository.",
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the repository exists under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the existing Vercel Project the repository belongs to.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
			},
		},
	}
}

func (d *vcrRepositoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config VCRRepository
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := d.client.GetVCRRepository(ctx, client.GetVCRRepositoryRequest{
		TeamID:    config.TeamID.ValueString(),
		ProjectID: config.ProjectID.ValueString(),
		Name:      config.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading VCR Repository",
			fmt.Sprintf("Could not read VCR Repository %s %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ProjectID.ValueString(),
				config.Name.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "read vcr repository", map[string]any{
		"team_id":    res.TeamID,
		"project_id": res.ProjectID,
		"name":       res.Name,
	})

	diags = resp.State.Set(ctx, convertResponseToVCRRepository(res))
	resp.Diagnostics.Append(diags...)
}
