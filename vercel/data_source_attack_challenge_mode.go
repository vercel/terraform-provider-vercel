package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &attackChallengeModeDataSource{}
	_ datasource.DataSourceWithConfigure = &attackChallengeModeDataSource{}
)

func newAttackChallengeModeDataSource() datasource.DataSource {
	return &attackChallengeModeDataSource{}
}

type attackChallengeModeDataSource struct {
	client *client.Client
}

func (d *attackChallengeModeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_attack_challenge_mode"
}

func (d *attackChallengeModeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *attackChallengeModeDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides an Attack Challenge Mode resource.

Attack Challenge Mode prevent malicious traffic by showing a verification challenge for every visitor.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The resource identifier.",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the Project to adjust the CPU for.",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the Project exists under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether Attack Challenge Mode is enabled or not.",
			},
		},
	}
}

func (d *attackChallengeModeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config AttackChallengeMode
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetAttackChallengeMode(ctx, config.ProjectID.ValueString(), config.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Attack Challenge Mode",
			fmt.Sprintf("Could not get Attack Challenge Mode %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	result := responseToAttackChallengeMode(out)
	tflog.Info(ctx, "read attack challenge mode", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
