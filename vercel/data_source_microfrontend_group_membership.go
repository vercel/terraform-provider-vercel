package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &microfrontendGroupMembershipDataSource{}
	_ datasource.DataSourceWithConfigure = &microfrontendGroupMembershipDataSource{}
)

func newMicrofrontendGroupMembershipDataSource() datasource.DataSource {
	return &microfrontendGroupMembershipDataSource{}
}

type microfrontendGroupMembershipDataSource struct {
	client *client.Client
}

func (d *microfrontendGroupMembershipDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_microfrontend_group_membership"
}

func (d *microfrontendGroupMembershipDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Schema returns the schema information for an microfrontendGroupMembership data source
func (r *microfrontendGroupMembershipDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Microfrontend Group Membership.

A Microfrontend Group Membership is a definition of a Vercel Project being a part of a Microfrontend Group.
`,
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The ID of the project.",
				Required:    true,
			},
			"microfrontend_group_id": schema.StringAttribute{
				Description: "The ID of the microfrontend group.",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Description: "The team ID to add the microfrontend group to. Required when configuring a team resource if a default team has not been set in the provider.",
				Optional:    true,
				Computed:    true,
			},
			"default_route": schema.StringAttribute{
				Description: "The default route for the project. Used for the screenshot of deployments.",
				Computed:    true,
			},
			"route_observability_to_this_project": schema.BoolAttribute{
				Description: "Whether the project is route observability for this project. If dalse, the project will be route observability for all projects to the default project.",
				Computed:    true,
			},
		},
	}
}

func (d *microfrontendGroupMembershipDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config MicrofrontendGroupMembership
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetMicrofrontendGroupMembership(ctx, config.TeamID.ValueString(),
		config.MicrofrontendGroupID.ValueString(),
		config.ProjectID.ValueString(),
	)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading microfrontend group membership",
			fmt.Sprintf("Could not get microfrontend group %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.MicrofrontendGroupID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToMicrofrontendGroupMembership(out)
	tflog.Info(ctx, "read microfrontend group membership", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"group_id":   result.MicrofrontendGroupID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
