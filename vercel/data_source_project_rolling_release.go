package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
				Description: "The ID of the project.",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the Vercel team.",
			},
			"advancement_type": schema.StringAttribute{
				Description: "The type of advancement for the rolling release. Either 'automatic' or 'manual-approval'.",
				Computed:    true,
			},
			"stages": schema.ListNestedAttribute{
				Description: "The stages for the rolling release configuration.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"target_percentage": schema.Int64Attribute{
							Description: "The percentage of traffic to route to this stage.",
							Computed:    true,
						},
						"duration": schema.Int64Attribute{
							Description: "The duration in minutes to wait before advancing to the next stage. Present for automatic advancement type.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// ProjectRollingReleaseDataSourceModel reflects the structure of the data source.
type ProjectRollingReleaseDataSourceModel struct {
	AdvancementType types.String `tfsdk:"advancement_type"`
	Stages          types.List   `tfsdk:"stages"`
	ProjectID       types.String `tfsdk:"project_id"`
	TeamID          types.String `tfsdk:"team_id"`
}

func (d *projectRollingReleaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProjectRollingReleaseDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var out client.RollingReleaseInfo
	var err error
	var convertedData ProjectRollingReleaseDataSourceModel
	var diags diag.Diagnostics

	out, err = d.client.GetRollingRelease(ctx, data.ProjectID.ValueString(), data.TeamID.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project rolling release",
			fmt.Sprintf("Could not get project rolling release %s %s, unexpected error: %s",
				data.TeamID.ValueString(),
				data.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	convertedData, diags = convertResponseToRollingReleaseDataSourceWithConfig(out, data, ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "converted data before setting state", map[string]any{
		"project_id":       convertedData.ProjectID.ValueString(),
		"team_id":          convertedData.TeamID.ValueString(),
		"advancement_type": convertedData.AdvancementType.ValueString(),
		"stages_is_null":   convertedData.Stages.IsNull(),
		"stages_length":    len(convertedData.Stages.Elements()),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &convertedData)...)
}

func convertResponseToRollingReleaseDataSourceWithConfig(response client.RollingReleaseInfo, config ProjectRollingReleaseDataSourceModel, ctx context.Context) (ProjectRollingReleaseDataSourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	result := ProjectRollingReleaseDataSourceModel{
		ProjectID: types.StringValue(response.ProjectID),
		TeamID:    types.StringValue(response.TeamID),
	}

	// Always initialize advancement_type and stages, even if null
	result.AdvancementType = types.StringValue("")
	result.Stages = types.ListValueMust(RollingReleaseStageElementType, []attr.Value{})

	// If API has no stages, return empty values
	if len(response.RollingRelease.Stages) == 0 {
		tflog.Info(ctx, "API has no stages, returning empty values")
		return result, diags
	}

	// Infer advancement_type if not set
	advancementType := response.RollingRelease.AdvancementType
	if advancementType == "" {
		hasDuration := false
		for _, stage := range response.RollingRelease.Stages {
			if stage.Duration != nil {
				hasDuration = true
				break
			}
		}
		if hasDuration {
			advancementType = "automatic"
		} else {
			advancementType = "manual-approval"
		}
		tflog.Info(ctx, "determined advancement type", map[string]any{
			"determined_advancement_type": advancementType,
			"has_duration":                hasDuration,
		})
	}
	result.AdvancementType = types.StringValue(advancementType)

	// Map stages (excluding terminal stage)
	var rollingReleaseStages []RollingReleaseStage
	for _, stage := range response.RollingRelease.Stages {
		if stage.TargetPercentage == 100 {
			continue
		}
		rollingReleaseStage := RollingReleaseStage{
			TargetPercentage: types.Int64Value(int64(stage.TargetPercentage)),
		}
		if stage.Duration != nil {
			rollingReleaseStage.Duration = types.Int64Value(int64(*stage.Duration))
		}
		rollingReleaseStages = append(rollingReleaseStages, rollingReleaseStage)
	}
	tflog.Info(ctx, "converted stages", map[string]any{
		"rolling_release_stages_count": len(rollingReleaseStages),
	})
	stages := make([]attr.Value, len(rollingReleaseStages))
	for i, stage := range rollingReleaseStages {
		stageObj := types.ObjectValueMust(
			RollingReleaseStageElementType.AttrTypes,
			map[string]attr.Value{
				"target_percentage": stage.TargetPercentage,
				"duration":          stage.Duration,
			},
		)
		stages[i] = stageObj
	}
	stagesList := types.ListValueMust(RollingReleaseStageElementType, stages)
	result.Stages = stagesList
	tflog.Info(ctx, "converted rolling release response", map[string]any{
		"advancement_type":        advancementType,
		"stages_count":            len(response.RollingRelease.Stages),
		"enabled":                 response.RollingRelease.Enabled,
		"result_advancement_type": result.AdvancementType.ValueString(),
		"result_stages_is_null":   result.Stages.IsNull(),
		"result_stages_length":    len(result.Stages.Elements()),
	})
	return result, diags
}
