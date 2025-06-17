package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

var (
	_ datasource.DataSource = &projectRollingReleaseDataSource{}
)

func newProjectRollingReleaseDataSource() datasource.DataSource {
	return &projectRollingReleaseDataSource{}
}

type projectRollingReleaseDataSource struct {
	client *client.Client
}

func (d *projectRollingReleaseDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_rolling_release"
}

func (d *projectRollingReleaseDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *projectRollingReleaseDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Data source for a Vercel project rolling release configuration.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project.",
				Required:            true,
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the Vercel team.",
			},
			"rolling_release": schema.SingleNestedAttribute{
				MarkdownDescription: "The rolling release configuration.",
				Computed:            true,
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether rolling releases are enabled.",
						Computed:            true,
					},
					"advancement_type": schema.StringAttribute{
						MarkdownDescription: "The type of advancement between stages. Must be either 'automatic' or 'manual-approval'. Required when enabled is true.",
						Computed:            true,
					},
					"stages": schema.ListNestedAttribute{
						MarkdownDescription: "The stages of the rolling release. Required when enabled is true.",
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"target_percentage": schema.Int64Attribute{
									MarkdownDescription: "The percentage of traffic to route to this stage.",
									Computed:            true,
								},
								"duration": schema.Int64Attribute{
									MarkdownDescription: "The duration in minutes to wait before advancing to the next stage. Required for all stages except the final stage when using automatic advancement.",
									Optional:            true,
								},
								"require_approval": schema.BoolAttribute{
									MarkdownDescription: "Whether approval is required before advancing to the next stage.",
									Computed:            true,
								},
							},
						},
					},
				},
			},
		},
	}
}

type RollingReleaseStageDataSource struct {
	TargetPercentage types.Int64 `tfsdk:"target_percentage"`
	Duration         types.Int64 `tfsdk:"duration"`
	RequireApproval  types.Bool  `tfsdk:"require_approval"`
}

type RollingReleaseDataSource struct {
	Enabled         types.Bool                      `tfsdk:"enabled"`
	AdvancementType types.String                    `tfsdk:"advancement_type"`
	Stages          []RollingReleaseStageDataSource `tfsdk:"stages"`
}

type RollingReleaseInfoDataSource struct {
	RollingRelease types.Object `tfsdk:"rolling_release"`
	ProjectID      types.String `tfsdk:"project_id"`
	TeamID         types.String `tfsdk:"team_id"`
}

func convertStagesDataSource(stages []client.RollingReleaseStage) []RollingReleaseStageDataSource {
	if len(stages) == 0 {
		return []RollingReleaseStageDataSource{}
	}

	result := make([]RollingReleaseStageDataSource, len(stages))
	for i, stage := range stages {
		duration := types.Int64Null()
		if stage.Duration != nil {
			duration = types.Int64Value(int64(*stage.Duration))
		}

		result[i] = RollingReleaseStageDataSource{
			TargetPercentage: types.Int64Value(int64(stage.TargetPercentage)),
			Duration:         duration,
			RequireApproval:  types.BoolValue(stage.RequireApproval),
		}
	}
	return result
}

func convertResponseToRollingReleaseDataSource(response client.RollingReleaseInfo) RollingReleaseInfoDataSource {
	rollingRelease := RollingReleaseDataSource{
		Enabled:         types.BoolValue(response.RollingRelease.Enabled),
		AdvancementType: types.StringValue(response.RollingRelease.AdvancementType),
		Stages:          convertStagesDataSource(response.RollingRelease.Stages),
	}

	if !response.RollingRelease.Enabled {
		rollingRelease.AdvancementType = types.StringValue("")
		rollingRelease.Stages = make([]RollingReleaseStageDataSource, 0)
	}

	rollingReleaseObj, _ := types.ObjectValueFrom(context.Background(), map[string]attr.Type{
		"enabled":          types.BoolType,
		"advancement_type": types.StringType,
		"stages": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
			"target_percentage": types.Int64Type,
			"duration":          types.Int64Type,
			"require_approval":  types.BoolType,
		}}},
	}, rollingRelease)

	return RollingReleaseInfoDataSource{
		RollingRelease: rollingReleaseObj,
		ProjectID:      types.StringValue(response.ProjectID),
		TeamID:         types.StringValue(response.TeamID),
	}
}

func (d *projectRollingReleaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config RollingReleaseInfoDataSource
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetRollingRelease(ctx, config.ProjectID.ValueString(), config.TeamID.ValueString())
	if client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Error reading project rolling release",
			fmt.Sprintf("No project rolling release found with id %s %s", config.TeamID.ValueString(), config.ProjectID.ValueString()),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project rolling release",
			fmt.Sprintf("Could not get project rolling release %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToRollingReleaseDataSource(out)
	tflog.Info(ctx, "read project rolling release", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
