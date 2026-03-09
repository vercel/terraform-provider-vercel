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
	_ datasource.DataSource              = &featureFlagSDKKeyDataSource{}
	_ datasource.DataSourceWithConfigure = &featureFlagSDKKeyDataSource{}
)

func newFeatureFlagSDKKeyDataSource() datasource.DataSource {
	return &featureFlagSDKKeyDataSource{}
}

type featureFlagSDKKeyDataSource struct {
	client *client.Client
}

type featureFlagSDKKeyDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	ProjectID   types.String `tfsdk:"project_id"`
	TeamID      types.String `tfsdk:"team_id"`
	Environment types.String `tfsdk:"environment"`
	Type        types.String `tfsdk:"type"`
	Label       types.String `tfsdk:"label"`
}

func (d *featureFlagSDKKeyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature_flag_sdk_key"
}

func (d *featureFlagSDKKeyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *featureFlagSDKKeyDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Feature Flag SDK Key.

Vercel only returns the cleartext SDK key and connection string at creation time, so this data source exposes lookup metadata only.
`,
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the Vercel team. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the Vercel project that owns the SDK key.",
			},
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The hash key identifier for the SDK key.",
			},
			"environment": schema.StringAttribute{
				Computed:    true,
				Description: "The environment this SDK key authenticates against.",
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "The SDK key type.",
			},
			"label": schema.StringAttribute{
				Computed:    true,
				Description: "An optional label used to identify this key in the Vercel dashboard.",
			},
		},
	}
}

func (d *featureFlagSDKKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config featureFlagSDKKeyDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.TeamID = types.StringValue(d.client.TeamID(config.TeamID.ValueString()))

	keys, err := d.client.ListFeatureFlagSDKKeys(ctx, client.ListFeatureFlagSDKKeysRequest{
		ProjectID: config.ProjectID.ValueString(),
		TeamID:    config.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Feature Flag SDK Key",
			fmt.Sprintf(
				"Could not list Feature Flag SDK Keys for %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	for _, key := range keys {
		if key.HashKey != config.ID.ValueString() {
			continue
		}

		result := featureFlagSDKKeyDataSourceModel{
			ID:          types.StringValue(key.HashKey),
			ProjectID:   types.StringValue(key.ProjectID),
			TeamID:      config.TeamID,
			Environment: types.StringValue(key.Environment),
			Type:        types.StringValue(key.Type),
			Label:       types.StringValue(key.Label),
		}

		tflog.Info(ctx, "read feature flag sdk key data source", map[string]any{
			"project_id": result.ProjectID.ValueString(),
			"team_id":    result.TeamID.ValueString(),
			"hash_key":   result.ID.ValueString(),
		})

		diags = resp.State.Set(ctx, result)
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.AddError(
		"Feature Flag SDK Key not found",
		fmt.Sprintf(
			"Could not find Feature Flag SDK Key %s for project %s.",
			config.ID.ValueString(),
			config.ProjectID.ValueString(),
		),
	)
}
