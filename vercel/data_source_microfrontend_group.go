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
	_ datasource.DataSource              = &microfrontendGroupDataSource{}
	_ datasource.DataSourceWithConfigure = &microfrontendGroupDataSource{}
)

func newMicrofrontendGroupDataSource() datasource.DataSource {
	return &microfrontendGroupDataSource{}
}

type microfrontendGroupDataSource struct {
	client *client.Client
}

func (d *microfrontendGroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_microfrontend_group"
}

func (d *microfrontendGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Schema returns the schema information for an microfrontendGroup data source
func (r *microfrontendGroupDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Microfrontend Group.

A Microfrontend Group is a definition of a microfrontend belonging to a Vercel Team.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "A unique identifier for the group of microfrontends. Example: mfe_12HKQaOmR5t5Uy6vdcQsNIiZgHGB",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "A human readable name for the microfrontends group.",
				Computed:    true,
			},
			"slug": schema.StringAttribute{
				Description: "A slugified version of the name.",
				Computed:    true,
			},
			"team_id": schema.StringAttribute{
				Description: "The team ID to add the project to. Required when configuring a team resource if a default team has not been set in the provider.",
				Optional:    true,
				Computed:    true,
			},
			"default_app": schema.SingleNestedAttribute{
				Description: "The default app for the project. Used as the entry point for the microfrontend.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"project_id": schema.StringAttribute{
						Description: "The ID of the project.",
						Computed:    true,
					},
					"default_route": schema.StringAttribute{
						Description: "The default route for the project. Used for the screenshot of deployments.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (d *microfrontendGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Info(ctx, "Reading microfrontend group")
	var config MicrofrontendGroup
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetMicrofrontendGroup(ctx, config.ID.ValueString(), config.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading microfrontend group",
			fmt.Sprintf("Could not get microfrontend group %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToMicrofrontendGroup(out)
	tflog.Info(ctx, "read microfrontend group", map[string]any{
		"result": result,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
