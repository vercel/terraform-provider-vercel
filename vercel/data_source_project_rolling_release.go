package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
			"automatic_rolling_release": schema.ListNestedAttribute{
				MarkdownDescription: "Automatic rolling release configuration.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"target_percentage": schema.Int64Attribute{
							MarkdownDescription: "The percentage of traffic to route to this stage.",
							Required:            true,
							Validators: []validator.Int64{
								int64validator.Between(0, 100),
							},
						},
						"duration": schema.Int64Attribute{
							MarkdownDescription: "The duration in minutes to wait before advancing to the next stage.",
							Required:            true,
							Validators: []validator.Int64{
								int64validator.Between(1, 10000),
							},
						},
					},
				},
			},
			"manual_rolling_release": schema.ListNestedAttribute{
				MarkdownDescription: "Manual rolling release configuration.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"target_percentage": schema.Int64Attribute{
							MarkdownDescription: "The percentage of traffic to route to this stage.",
							Required:            true,
							Validators: []validator.Int64{
								int64validator.Between(0, 100),
							},
						},
					},
				},
			},
		},
	}
}

// ProjectRollingReleaseDataSourceModel reflects the structure of the data source.
type ProjectRollingReleaseDataSourceModel struct {
	AutomaticRollingRelease types.List   `tfsdk:"automatic_rolling_release"`
	ManualRollingRelease    types.List   `tfsdk:"manual_rolling_release"`
	ProjectID               types.String `tfsdk:"project_id"`
	TeamID                  types.String `tfsdk:"team_id"`
}

func (d *projectRollingReleaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProjectRollingReleaseDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetRollingRelease(ctx, data.ProjectID.ValueString(), data.TeamID.ValueString())
	if client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Error reading project rolling release",
			fmt.Sprintf("No project rolling release found with id %s %s", data.TeamID.ValueString(), data.ProjectID.ValueString()),
		)
		return
	}
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

	// Convert the response to the data source model
	convertedData, diags := convertResponseToRollingReleaseDataSource(out, ctx)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &convertedData)...)
}

func convertResponseToRollingReleaseDataSource(response client.RollingReleaseInfo, ctx context.Context) (ProjectRollingReleaseDataSourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Log the raw response for debugging
	tflog.Info(ctx, "raw rolling release response", map[string]any{
		"project_id":       response.ProjectID,
		"team_id":          response.TeamID,
		"enabled":          response.RollingRelease.Enabled,
		"advancement_type": response.RollingRelease.AdvancementType,
		"stages_count":     len(response.RollingRelease.Stages),
		"stages":           response.RollingRelease.Stages,
	})

	result := ProjectRollingReleaseDataSourceModel{
		ProjectID: types.StringValue(response.ProjectID),
		TeamID:    types.StringValue(response.TeamID),
	}
	// Initialize empty lists for both types
	result.AutomaticRollingRelease = types.ListValueMust(AutomaticRollingReleaseElementType, []attr.Value{})
	result.ManualRollingRelease = types.ListValueMust(ManualRollingReleaseElementType, []attr.Value{})

	// If no stages, return empty lists
	if len(response.RollingRelease.Stages) == 0 {
		return result, diags
	}

	// Determine which type of rolling release to use based on API response
	// If advancementType is empty but stages exist, determine type from stage properties
	advancementType := response.RollingRelease.AdvancementType
	tflog.Info(ctx, "determining advancement type", map[string]any{
		"original_advancement_type": advancementType,
		"stages_count":              len(response.RollingRelease.Stages),
		"enabled":                   response.RollingRelease.Enabled,
	})

	if advancementType == "" && len(response.RollingRelease.Stages) > 0 {
		// Check if stages have duration (automatic) or not (manual)
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

	if advancementType == "automatic" {
		// Convert API stages to automatic stages (excluding terminal stage)
		var automaticStages []AutomaticStage
		for _, stage := range response.RollingRelease.Stages {
			// Skip the terminal stage (100%)
			if stage.TargetPercentage == 100 {
				continue
			}

			var duration types.Int64
			if stage.Duration != nil {
				duration = types.Int64Value(int64(*stage.Duration))
			} else {
				duration = types.Int64Value(60) // Default duration
			}

			automaticStages = append(automaticStages, AutomaticStage{
				TargetPercentage: types.Int64Value(int64(stage.TargetPercentage)),
				Duration:         duration,
			})
		}

		// Convert to Terraform types
		stages := make([]attr.Value, len(automaticStages))
		for i, stage := range automaticStages {
			stageObj := types.ObjectValueMust(
				AutomaticRollingReleaseElementType.AttrTypes,
				map[string]attr.Value{
					"target_percentage": stage.TargetPercentage,
					"duration":          stage.Duration,
				},
			)
			stages[i] = stageObj
		}

		stagesList := types.ListValueMust(AutomaticRollingReleaseElementType, stages)
		result.AutomaticRollingRelease = stagesList

	} else if advancementType == "manual-approval" {
		// Convert API stages to manual stages (excluding terminal stage)
		var manualStages []ManualStage
		tflog.Info(ctx, "processing manual stages", map[string]any{
			"total_stages": len(response.RollingRelease.Stages),
		})

		for i, stage := range response.RollingRelease.Stages {
			tflog.Info(ctx, "processing stage", map[string]any{
				"stage_index":       i,
				"target_percentage": stage.TargetPercentage,
				"require_approval":  stage.RequireApproval,
				"duration":          stage.Duration,
			})

			// Skip the terminal stage (100%)
			if stage.TargetPercentage == 100 {
				tflog.Info(ctx, "skipping terminal stage", map[string]any{
					"stage_index": i,
				})
				continue
			}

			manualStages = append(manualStages, ManualStage{
				TargetPercentage: types.Int64Value(int64(stage.TargetPercentage)),
			})
		}

		tflog.Info(ctx, "manual stages after filtering", map[string]any{
			"manual_stages_count": len(manualStages),
		})

		// Convert to Terraform types
		stages := make([]attr.Value, len(manualStages))
		for i, stage := range manualStages {
			stageObj := types.ObjectValueMust(
				ManualRollingReleaseElementType.AttrTypes,
				map[string]attr.Value{
					"target_percentage": stage.TargetPercentage,
				},
			)
			stages[i] = stageObj
		}

		stagesList := types.ListValueMust(ManualRollingReleaseElementType, stages)
		tflog.Info(ctx, "created manual stages list", map[string]any{
			"stages_count": len(stages),
			"stages_list":  stagesList,
			"is_null":      stagesList.IsNull(),
			"is_unknown":   stagesList.IsUnknown(),
		})
		result.ManualRollingRelease = stagesList

		tflog.Info(ctx, "final manual rolling release result", map[string]any{
			"stages_count": len(stages),
			"is_null":      result.ManualRollingRelease.IsNull(),
			"is_unknown":   result.ManualRollingRelease.IsUnknown(),
			"stages_list":  stagesList,
		})
	}

	// Log the conversion result for debugging
	tflog.Info(ctx, "converted rolling release response", map[string]any{
		"original_advancement_type":         response.RollingRelease.AdvancementType,
		"determined_advancement_type":       advancementType,
		"stages_count":                      len(response.RollingRelease.Stages),
		"enabled":                           response.RollingRelease.Enabled,
		"automatic_rolling_release_is_null": result.AutomaticRollingRelease.IsNull(),
		"manual_rolling_release_is_null":    result.ManualRollingRelease.IsNull(),
		"automatic_rolling_release_unknown": result.AutomaticRollingRelease.IsUnknown(),
		"manual_rolling_release_unknown":    result.ManualRollingRelease.IsUnknown(),
	})

	return result, diags
}
