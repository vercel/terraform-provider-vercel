package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var (
	_ datasource.DataSource              = &featureFlagDataSource{}
	_ datasource.DataSourceWithConfigure = &featureFlagDataSource{}
)

func newFeatureFlagDataSource() datasource.DataSource {
	return &featureFlagDataSource{}
}

type featureFlagDataSource struct {
	client *client.Client
}

type featureFlagDataSourceModel struct {
	ID          types.String                 `tfsdk:"id"`
	ProjectID   types.String                 `tfsdk:"project_id"`
	TeamID      types.String                 `tfsdk:"team_id"`
	Key         types.String                 `tfsdk:"key"`
	Description types.String                 `tfsdk:"description"`
	Kind        types.String                 `tfsdk:"kind"`
	Archived    types.Bool                   `tfsdk:"archived"`
	Variant     []featureFlagVariantModel    `tfsdk:"variant"`
	Production  *featureFlagEnvironmentModel `tfsdk:"production"`
	Preview     *featureFlagEnvironmentModel `tfsdk:"preview"`
	Development *featureFlagEnvironmentModel `tfsdk:"development"`
}

func (d *featureFlagDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature_flag"
}

func (d *featureFlagDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func featureFlagDataSourceEnvironmentSchema(description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Computed:    true,
		Description: description,
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the flag actively evaluates in this environment.",
			},
			"default_variant_id": schema.StringAttribute{
				Computed:    true,
				Description: "The variant served when this environment is enabled and no rules match.",
			},
			"disabled_variant_id": schema.StringAttribute{
				Computed:    true,
				Description: "The variant served while this environment is disabled or paused.",
			},
		},
	}
}

func (d *featureFlagDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Feature Flag.

This data source reads the simplified static flag shape exposed by this provider and looks up the flag by its stable ` + "`key`" + ` within a project.
`,
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the Vercel team. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the Vercel project that owns the flag.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the feature flag.",
			},
			"key": schema.StringAttribute{
				Required:    true,
				Description: "The stable flag key used in your application code.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						featureFlagKeyRegex,
						"Flag keys may only contain letters, numbers, dashes, and underscores.",
					),
				},
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "A human-readable description of the flag.",
			},
			"kind": schema.StringAttribute{
				Computed:    true,
				Description: "The type of value this flag returns.",
			},
			"archived": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the flag is archived.",
			},
			"variant": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The variants available for this flag.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The stable variant identifier.",
						},
						"label": schema.StringAttribute{
							Computed:    true,
							Description: "A human-readable label for the variant.",
						},
						"description": schema.StringAttribute{
							Computed:    true,
							Description: "A human-readable description for the variant.",
						},
						"value_string": schema.StringAttribute{
							Computed:    true,
							Description: "The string value for this variant when `kind = \"string\"`.",
						},
						"value_number": schema.NumberAttribute{
							Computed:    true,
							Description: "The numeric value for this variant when `kind = \"number\"`.",
						},
						"value_bool": schema.BoolAttribute{
							Computed:    true,
							Description: "The boolean value for this variant when `kind = \"boolean\"`.",
						},
					},
				},
			},
			"production":  featureFlagDataSourceEnvironmentSchema("The production environment behavior for this flag."),
			"preview":     featureFlagDataSourceEnvironmentSchema("The preview environment behavior for this flag."),
			"development": featureFlagDataSourceEnvironmentSchema("The development environment behavior for this flag."),
		},
	}
}

func (d *featureFlagDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config featureFlagDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.TeamID = types.StringValue(d.client.TeamID(config.TeamID.ValueString()))

	out, err := d.client.GetFeatureFlag(ctx, client.GetFeatureFlagRequest{
		ProjectID:    config.ProjectID.ValueString(),
		TeamID:       config.TeamID.ValueString(),
		FlagIDOrSlug: config.Key.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Feature Flag",
			fmt.Sprintf(
				"Could not get Feature Flag %s %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ProjectID.ValueString(),
				config.Key.ValueString(),
				err,
			),
		)
		return
	}

	resourceModel, diags := featureFlagFromClient(out, featureFlagModel{
		ProjectID: config.ProjectID,
		TeamID:    config.TeamID,
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result := featureFlagDataSourceModelFromResource(resourceModel)
	tflog.Info(ctx, "read feature flag data source", map[string]any{
		"project_id": result.ProjectID.ValueString(),
		"team_id":    result.TeamID.ValueString(),
		"flag_id":    result.ID.ValueString(),
		"key":        result.Key.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func featureFlagDataSourceModelFromResource(in featureFlagModel) featureFlagDataSourceModel {
	return featureFlagDataSourceModel{
		ID:          in.ID,
		ProjectID:   in.ProjectID,
		TeamID:      in.TeamID,
		Key:         in.Key,
		Description: in.Description,
		Kind:        in.Kind,
		Archived:    in.Archived,
		Variant:     in.Variant,
		Production:  &in.Production,
		Preview:     &in.Preview,
		Development: &in.Development,
	}
}
