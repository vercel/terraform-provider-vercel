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
	_ datasource.DataSource              = &featureFlagSegmentDataSource{}
	_ datasource.DataSourceWithConfigure = &featureFlagSegmentDataSource{}
)

func newFeatureFlagSegmentDataSource() datasource.DataSource {
	return &featureFlagSegmentDataSource{}
}

type featureFlagSegmentDataSource struct {
	client *client.Client
}

type featureFlagSegmentDataSourceModel struct {
	ID          types.String                   `tfsdk:"id"`
	ProjectID   types.String                   `tfsdk:"project_id"`
	TeamID      types.String                   `tfsdk:"team_id"`
	Slug        types.String                   `tfsdk:"slug"`
	Name        types.String                   `tfsdk:"name"`
	Description types.String                   `tfsdk:"description"`
	Hint        types.String                   `tfsdk:"hint"`
	Include     []featureFlagSegmentMatchModel `tfsdk:"include"`
	Exclude     []featureFlagSegmentMatchModel `tfsdk:"exclude"`
}

func (d *featureFlagSegmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature_flag_segment"
}

func (d *featureFlagSegmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func featureFlagSegmentDataSourceMatchSchema(description string) schema.SetNestedAttribute {
	return schema.SetNestedAttribute{
		Computed:    true,
		Description: description,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"entity": schema.StringAttribute{
					Computed:    true,
					Description: "The entity type to match.",
				},
				"attribute": schema.StringAttribute{
					Computed:    true,
					Description: "The entity attribute to match.",
				},
				"values": schema.SetAttribute{
					Computed:    true,
					ElementType: types.StringType,
					Description: "The exact values to include or exclude for this entity attribute.",
				},
			},
		},
	}
}

func (d *featureFlagSegmentDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Feature Flag Segment.

This data source reads the simplified exact-match segment shape exposed by this provider and looks up the segment by its stable ` + "`slug`" + ` within a project.
`,
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the Vercel team. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the Vercel project that owns the segment.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the segment.",
			},
			"slug": schema.StringAttribute{
				Required:    true,
				Description: "The stable segment slug used by the Vercel Flags API.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						featureFlagSegmentSlugRegex,
						"Segment slugs may only contain letters, numbers, dashes, and underscores.",
					),
				},
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The human-readable segment name shown in the Vercel dashboard.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "A human-readable description of the segment.",
			},
			"hint": schema.StringAttribute{
				Computed:    true,
				Description: "An optional dashboard hint for the segment.",
			},
			"include": featureFlagSegmentDataSourceMatchSchema("Exact entity attribute values that are always part of this segment."),
			"exclude": featureFlagSegmentDataSourceMatchSchema("Exact entity attribute values that are always excluded from this segment."),
		},
	}
}

func (d *featureFlagSegmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config featureFlagSegmentDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.TeamID = types.StringValue(d.client.TeamID(config.TeamID.ValueString()))

	out, err := d.client.GetFeatureFlagSegment(ctx, client.GetFeatureFlagSegmentRequest{
		ProjectID:       config.ProjectID.ValueString(),
		TeamID:          config.TeamID.ValueString(),
		SegmentIDOrSlug: config.Slug.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Feature Flag Segment",
			fmt.Sprintf(
				"Could not get Feature Flag Segment %s %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ProjectID.ValueString(),
				config.Slug.ValueString(),
				err,
			),
		)
		return
	}

	resourceModel, diags := featureFlagSegmentFromClient(ctx, out, featureFlagSegmentModel{
		ProjectID: config.ProjectID,
		TeamID:    config.TeamID,
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result := featureFlagSegmentDataSourceModelFromResource(resourceModel)
	tflog.Info(ctx, "read feature flag segment data source", map[string]any{
		"project_id": result.ProjectID.ValueString(),
		"team_id":    result.TeamID.ValueString(),
		"segment_id": result.ID.ValueString(),
		"slug":       result.Slug.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func featureFlagSegmentDataSourceModelFromResource(in featureFlagSegmentModel) featureFlagSegmentDataSourceModel {
	return featureFlagSegmentDataSourceModel(in)
}
