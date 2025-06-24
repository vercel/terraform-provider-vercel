package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

var (
	_ resource.Resource                = &projectRollingReleaseResource{}
	_ resource.ResourceWithConfigure   = &projectRollingReleaseResource{}
	_ resource.ResourceWithImportState = &projectRollingReleaseResource{}
)

func newProjectRollingReleaseResource() resource.Resource {
	return &projectRollingReleaseResource{}
}

type projectRollingReleaseResource struct {
	client *client.Client
}

func (r *projectRollingReleaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_rolling_release"
}

func (r *projectRollingReleaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = client
}

// Schema returns the schema information for a project rolling release resource.
func (r *projectRollingReleaseResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages rolling release configuration for a Vercel project.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project.",
				Required:            true,
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the Vercel team.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
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

type AutomaticStage struct {
	TargetPercentage types.Int64 `tfsdk:"target_percentage"`
	Duration         types.Int64 `tfsdk:"duration"`
}

type ManualStage struct {
	TargetPercentage types.Int64 `tfsdk:"target_percentage"`
}

// ProjectRollingRelease reflects the state terraform stores internally for a project rolling release.
type RollingReleaseInfo struct {
	AutomaticRollingRelease types.List   `tfsdk:"automatic_rolling_release"`
	ManualRollingRelease    types.List   `tfsdk:"manual_rolling_release"`
	ProjectID               types.String `tfsdk:"project_id"`
	TeamID                  types.String `tfsdk:"team_id"`
}

func (e *RollingReleaseInfo) toUpdateRollingReleaseRequest() (client.UpdateRollingReleaseRequest, diag.Diagnostics) {
	var stages []client.RollingReleaseStage
	var advancementType string
	var diags diag.Diagnostics

	if !e.AutomaticRollingRelease.IsNull() && !e.AutomaticRollingRelease.IsUnknown() {
		advancementType = "automatic"

		// Convert automatic stages
		var tfStages []AutomaticStage
		diags = e.AutomaticRollingRelease.ElementsAs(context.Background(), &tfStages, false)
		if diags.HasError() {
			return client.UpdateRollingReleaseRequest{}, diags
		}

		// Add all stages from config
		stages = make([]client.RollingReleaseStage, len(tfStages))
		for i, stage := range tfStages {
			duration := int(stage.Duration.ValueInt64())
			stages[i] = client.RollingReleaseStage{
				TargetPercentage: int(stage.TargetPercentage.ValueInt64()),
				Duration:         &duration,
				RequireApproval:  false,
			}
		}

		// Add terminal stage (100%) without duration
		stages = append(stages, client.RollingReleaseStage{
			TargetPercentage: 100,
			RequireApproval:  false,
		})

	} else if !e.ManualRollingRelease.IsNull() && !e.ManualRollingRelease.IsUnknown() {
		advancementType = "manual-approval"

		// Convert manual stages
		var tfStages []ManualStage
		diags = e.ManualRollingRelease.ElementsAs(context.Background(), &tfStages, false)
		if diags.HasError() {
			return client.UpdateRollingReleaseRequest{}, diags
		}

		// Add all stages from config
		stages = make([]client.RollingReleaseStage, len(tfStages))
		for i, stage := range tfStages {
			stages[i] = client.RollingReleaseStage{
				TargetPercentage: int(stage.TargetPercentage.ValueInt64()),
				RequireApproval:  true,
			}
		}

		// Add terminal stage (100%) without approval
		stages = append(stages, client.RollingReleaseStage{
			TargetPercentage: 100,
			RequireApproval:  false,
		})
	}

	// Log the request for debugging
	tflog.Info(context.Background(), "converting to update request", map[string]any{
		"advancement_type": advancementType,
		"stages_count":     len(stages),
	})

	return client.UpdateRollingReleaseRequest{
		RollingRelease: client.RollingRelease{
			Enabled:         true,
			AdvancementType: advancementType,
			Stages:          stages,
		},
		ProjectID: e.ProjectID.ValueString(),
		TeamID:    e.TeamID.ValueString(),
	}, diags
}

func convertResponseToRollingRelease(response client.RollingReleaseInfo, plan *RollingReleaseInfo, ctx context.Context) (RollingReleaseInfo, diag.Diagnostics) {
	var diags diag.Diagnostics

	result := RollingReleaseInfo{
		ProjectID: types.StringValue(response.ProjectID),
		TeamID:    types.StringValue(response.TeamID),
	}

	// If disabled, return empty values
	if !response.RollingRelease.Enabled {
		return result, diags
	}

	// Determine which type of rolling release to use based on API response
	if response.RollingRelease.AdvancementType == "automatic" {
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
				map[string]attr.Type{
					"target_percentage": types.Int64Type,
					"duration":          types.Int64Type,
				},
				map[string]attr.Value{
					"target_percentage": stage.TargetPercentage,
					"duration":          stage.Duration,
				},
			)
			stages[i] = stageObj
		}

		stagesList, stagesDiags := types.ListValueFrom(ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"target_percentage": types.Int64Type,
				"duration":          types.Int64Type,
			},
		}, stages)
		diags.Append(stagesDiags...)
		if diags.HasError() {
			return result, diags
		}

		result.AutomaticRollingRelease = stagesList

	} else if response.RollingRelease.AdvancementType == "manual-approval" {
		// Convert API stages to manual stages (excluding terminal stage)
		var manualStages []ManualStage
		for _, stage := range response.RollingRelease.Stages {
			// Skip the terminal stage (100%)
			if stage.TargetPercentage == 100 {
				continue
			}

			manualStages = append(manualStages, ManualStage{
				TargetPercentage: types.Int64Value(int64(stage.TargetPercentage)),
			})
		}

		// Convert to Terraform types
		stages := make([]attr.Value, len(manualStages))
		for i, stage := range manualStages {
			stageObj := types.ObjectValueMust(
				map[string]attr.Type{
					"target_percentage": types.Int64Type,
				},
				map[string]attr.Value{
					"target_percentage": stage.TargetPercentage,
				},
			)
			stages[i] = stageObj
		}

		stagesList, stagesDiags := types.ListValueFrom(ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"target_percentage": types.Int64Type,
			},
		}, stages)
		diags.Append(stagesDiags...)
		if diags.HasError() {
			return result, diags
		}

		result.ManualRollingRelease = stagesList
	}

	// Log the conversion result for debugging
	tflog.Info(ctx, "converted rolling release response", map[string]any{
		"advancement_type": response.RollingRelease.AdvancementType,
		"stages_count":     len(response.RollingRelease.Stages),
	})

	return result, diags
}

// Create will create a rolling release for a Vercel project by sending a request to the Vercel API.
func (r *projectRollingReleaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RollingReleaseInfo
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert plan to client request
	request, diags := plan.toUpdateRollingReleaseRequest()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Log the request for debugging
	tflog.Info(ctx, "creating rolling release", map[string]any{
		"enabled":          request.RollingRelease.Enabled,
		"advancement_type": request.RollingRelease.AdvancementType,
		"stages":           request.RollingRelease.Stages,
	})

	out, err := r.client.UpdateRollingRelease(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project rolling release",
			fmt.Sprintf("Could not create project rolling release, unexpected error: %s",
				err,
			),
		)
		return
	}

	// Convert response to state
	result, diags := convertResponseToRollingRelease(out, &plan, ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Log the result for debugging
	tflog.Debug(ctx, "created rolling release", map[string]any{
		"project_id": result.ProjectID.ValueString(),
	})

	// Set state
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read a rolling release of a Vercel project by requesting it from the Vercel API, and will update terraform
// with this information.
func (r *projectRollingReleaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RollingReleaseInfo
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetRollingRelease(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project rolling release",
			fmt.Sprintf("Could not get project rolling release, unexpected error: %s",
				err,
			),
		)
		return
	}

	// Log the response for debugging
	tflog.Debug(ctx, "got rolling release from API", map[string]any{
		"enabled":          out.RollingRelease.Enabled,
		"advancement_type": out.RollingRelease.AdvancementType,
		"stages":           out.RollingRelease.Stages,
	})

	result, diags := convertResponseToRollingRelease(out, &state, ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Log the result for debugging
	tflog.Debug(ctx, "converted rolling release", map[string]any{
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update will update an existing rolling release to the latest information.
func (r *projectRollingReleaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RollingReleaseInfo
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state RollingReleaseInfo
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert plan to client request
	request, diags := plan.toUpdateRollingReleaseRequest()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Log the request for debugging
	tflog.Debug(ctx, "updating rolling release", map[string]any{
		"enabled":          request.RollingRelease.Enabled,
		"advancement_type": request.RollingRelease.AdvancementType,
		"stages":           request.RollingRelease.Stages,
	})

	out, err := r.client.UpdateRollingRelease(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project rolling release",
			fmt.Sprintf("Could not update project rolling release, unexpected error: %s",
				err,
			),
		)
		return
	}

	// Convert response to state
	result, diags := convertResponseToRollingRelease(out, &plan, ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Log the result for debugging
	tflog.Debug(ctx, "updated rolling release", map[string]any{
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete will delete an existing rolling release by disabling it.
func (r *projectRollingReleaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RollingReleaseInfo
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Disable rolling release
	request := client.UpdateRollingReleaseRequest{
		RollingRelease: client.RollingRelease{
			Enabled:         false,
			AdvancementType: "",
			Stages:          []client.RollingReleaseStage{},
		},
		ProjectID: state.ProjectID.ValueString(),
		TeamID:    state.TeamID.ValueString(),
	}

	_, err := r.client.UpdateRollingRelease(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project rolling release",
			fmt.Sprintf("Could not delete project rolling release, unexpected error: %s",
				err,
			),
		)
		return
	}
}

// ImportState takes an identifier and reads all the project rolling release information from the Vercel API.
// The results are then stored in terraform state.
func (r *projectRollingReleaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing project rolling release",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id\" or \"project_id\"", req.ID),
		)
	}

	out, err := r.client.GetRollingRelease(ctx, projectID, teamID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project rolling release",
			fmt.Sprintf("Could not get project rolling release %s %s, unexpected error: %s",
				teamID,
				projectID,
				err,
			),
		)
		return
	}

	// For import, we don't have any state to preserve
	result, diags := convertResponseToRollingRelease(out, nil, ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "imported project rolling release", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
