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
			"rolling_release": schema.SingleNestedAttribute{
				MarkdownDescription: "The rolling release configuration.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether rolling releases are enabled.",
						Required:            true,
					},
					"advancement_type": schema.StringAttribute{
						MarkdownDescription: "The type of advancement between stages. Must be either 'automatic' or 'manual-approval'. Required when enabled is true.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							advancementTypeValidator{},
						},
					},
					"stages": schema.ListNestedAttribute{
						MarkdownDescription: "The stages of the rolling release. Required when enabled is true.",
						Optional:            true,
						Computed:            true,
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
									MarkdownDescription: "The duration in minutes to wait before advancing to the next stage. Required for all stages except the final stage when using automatic advancement.",
									Optional:            true,
									Computed:            true,
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

type RollingReleaseStage struct {
	TargetPercentage types.Int64 `tfsdk:"target_percentage"`
	Duration         types.Int64 `tfsdk:"duration"`
	RequireApproval  types.Bool  `tfsdk:"require_approval"`
}

// RollingRelease reflects the state terraform stores internally for a project rolling release.
type RollingRelease struct {
	Enabled         types.Bool   `tfsdk:"enabled"`
	AdvancementType types.String `tfsdk:"advancement_type"`
	Stages          types.List   `tfsdk:"stages"`
}

// ProjectRollingRelease reflects the state terraform stores internally for a project rolling release.
type RollingReleaseInfo struct {
	RollingRelease RollingRelease `tfsdk:"rolling_release"`
	ProjectID      types.String   `tfsdk:"project_id"`
	TeamID         types.String   `tfsdk:"team_id"`
}

func (e *RollingReleaseInfo) toUpdateRollingReleaseRequest() (client.UpdateRollingReleaseRequest, diag.Diagnostics) {
	var stages []client.RollingReleaseStage
	var advancementType string
	var diags diag.Diagnostics

	if e.RollingRelease.Enabled.ValueBool() {
		advancementType = e.RollingRelease.AdvancementType.ValueString()

		// Convert stages from types.List to []client.RollingReleaseStage
		var tfStages []RollingReleaseStage
		diags = e.RollingRelease.Stages.ElementsAs(context.Background(), &tfStages, false)
		if diags.HasError() {
			return client.UpdateRollingReleaseRequest{}, diags
		}

		stages = make([]client.RollingReleaseStage, len(tfStages))
		for i, stage := range tfStages {
			if advancementType == "automatic" {
				// For automatic advancement, duration is required except for last stage
				if i < len(tfStages)-1 {
					// Non-last stage needs duration
					if stage.Duration.IsNull() {
						// Default duration for non-last stages
						duration := 60
						stages[i] = client.RollingReleaseStage{
							TargetPercentage: int(stage.TargetPercentage.ValueInt64()),
							Duration:         &duration,
							RequireApproval:  stage.RequireApproval.ValueBool(),
						}
					} else {
						duration := int(stage.Duration.ValueInt64())
						stages[i] = client.RollingReleaseStage{
							TargetPercentage: int(stage.TargetPercentage.ValueInt64()),
							Duration:         &duration,
							RequireApproval:  stage.RequireApproval.ValueBool(),
						}
					}
				} else {
					// Last stage should not have duration
					stages[i] = client.RollingReleaseStage{
						TargetPercentage: int(stage.TargetPercentage.ValueInt64()),
						RequireApproval:  stage.RequireApproval.ValueBool(),
					}
				}
			} else {
				// For manual approval, omit duration field completely
				stages[i] = client.RollingReleaseStage{
					TargetPercentage: int(stage.TargetPercentage.ValueInt64()),
					RequireApproval:  stage.RequireApproval.ValueBool(),
				}
			}
		}
	} else {
		// When disabled, don't send any stages or advancement type to the API
		stages = []client.RollingReleaseStage{}
		advancementType = ""
	}

	// Log the request for debugging
	tflog.Info(context.Background(), "converting to update request", map[string]any{
		"enabled":          e.RollingRelease.Enabled.ValueBool(),
		"advancement_type": advancementType,
		"stages_count":     len(stages),
	})

	return client.UpdateRollingReleaseRequest{
		RollingRelease: client.RollingRelease{
			Enabled:         e.RollingRelease.Enabled.ValueBool(),
			AdvancementType: advancementType,
			Stages:          stages,
		},
		ProjectID: e.ProjectID.ValueString(),
		TeamID:    e.TeamID.ValueString(),
	}, diags
}

func convertResponseToRollingRelease(response client.RollingReleaseInfo, plan *RollingReleaseInfo, ctx context.Context) (RollingReleaseInfo, diag.Diagnostics) {
	var diags diag.Diagnostics
	advancementType := types.StringNull()
	if plan.RollingRelease.Enabled.ValueBool() {
		advancementType = plan.RollingRelease.AdvancementType
	}
	result := RollingReleaseInfo{
		RollingRelease: RollingRelease{
			Enabled:         plan.RollingRelease.Enabled,
			AdvancementType: advancementType,
		},
		ProjectID: types.StringValue(response.ProjectID),
		TeamID:    types.StringValue(response.TeamID),
	}

	// If disabled, return empty values
	if !plan.RollingRelease.Enabled.ValueBool() {
		result.RollingRelease.AdvancementType = types.StringValue("")
		// Create an empty list instead of null
		emptyStages, stagesDiags := types.ListValueFrom(ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"target_percentage": types.Int64Type,
				"duration":          types.Int64Type,
				"require_approval":  types.BoolType,
			},
		}, []attr.Value{})
		diags.Append(stagesDiags...)
		if diags.HasError() {
			return result, diags
		}
		result.RollingRelease.Stages = emptyStages
		return result, diags
	}

	// If we have a plan, try to match stages by target percentage to preserve order
	var orderedStages []client.RollingReleaseStage
	if plan != nil && !plan.RollingRelease.Stages.IsNull() && !plan.RollingRelease.Stages.IsUnknown() {
		var planStages []RollingReleaseStage
		diags.Append(plan.RollingRelease.Stages.ElementsAs(ctx, &planStages, false)...)
		if diags.HasError() {
			return result, diags
		}

		// Create a map of target percentage to stage for quick lookup
		stageMap := make(map[int]client.RollingReleaseStage)
		for _, stage := range response.RollingRelease.Stages {
			stageMap[stage.TargetPercentage] = stage
		}

		// Try to preserve the order from the plan
		orderedStages = make([]client.RollingReleaseStage, 0, len(response.RollingRelease.Stages))
		for _, planStage := range planStages {
			if stage, ok := stageMap[int(planStage.TargetPercentage.ValueInt64())]; ok {
				orderedStages = append(orderedStages, stage)
				delete(stageMap, stage.TargetPercentage)
			}
		}

		// Add any remaining stages that weren't in the plan
		for _, stage := range response.RollingRelease.Stages {
			if _, ok := stageMap[stage.TargetPercentage]; ok {
				orderedStages = append(orderedStages, stage)
			}
		}
	} else {
		orderedStages = response.RollingRelease.Stages
	}

	// Convert stages from response
	elements := make([]attr.Value, len(orderedStages))
	for i, stage := range orderedStages {
		var duration types.Int64
		if response.RollingRelease.AdvancementType == "automatic" {
			// For automatic advancement, duration is required except for the last stage
			if i < len(orderedStages)-1 {
				if stage.Duration != nil {
					duration = types.Int64Value(int64(*stage.Duration))
				} else {
					duration = types.Int64Value(60) // Default duration in minutes
				}
			} else {
				duration = types.Int64Value(0) // Last stage doesn't need duration
			}
		} else {
			// For manual approval, duration is not used
			duration = types.Int64Null()
		}

		elements[i] = types.ObjectValueMust(
			map[string]attr.Type{
				"target_percentage": types.Int64Type,
				"duration":          types.Int64Type,
				"require_approval":  types.BoolType,
			},
			map[string]attr.Value{
				"target_percentage": types.Int64Value(int64(stage.TargetPercentage)),
				"duration":          duration,
				"require_approval":  types.BoolValue(stage.RequireApproval),
			},
		)
	}

	stages, stagesDiags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"target_percentage": types.Int64Type,
			"duration":          types.Int64Type,
			"require_approval":  types.BoolType,
		},
	}, elements)
	diags.Append(stagesDiags...)
	if diags.HasError() {
		return result, diags
	}
	result.RollingRelease.Stages = stages

	// Log the conversion result for debugging
	tflog.Info(ctx, "converted rolling release response", map[string]any{
		"enabled":          result.RollingRelease.Enabled.ValueBool(),
		"advancement_type": result.RollingRelease.AdvancementType.ValueString(),
		"stages_count":     len(elements),
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

	// If we're enabling, first create in disabled state then enable
	if request.RollingRelease.Enabled {
		// First create in disabled state
		disabledRequest := client.UpdateRollingReleaseRequest{
			RollingRelease: client.RollingRelease{
				Enabled:         false,
				AdvancementType: "",
				Stages:          []client.RollingReleaseStage{},
			},
			ProjectID: request.ProjectID,
			TeamID:    request.TeamID,
		}

		_, err := r.client.UpdateRollingRelease(ctx, disabledRequest)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating project rolling release",
				fmt.Sprintf("Could not create project rolling release in disabled state, unexpected error: %s",
					err,
				),
			)
			return
		}
	}

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
		"enabled":          result.RollingRelease.Enabled.ValueBool(),
		"advancement_type": result.RollingRelease.AdvancementType.ValueString(),
		"stages":           result.RollingRelease.Stages,
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
		"enabled":          result.RollingRelease.Enabled.ValueBool(),
		"advancement_type": result.RollingRelease.AdvancementType.ValueString(),
		"stages":           result.RollingRelease.Stages,
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

	// If we're transitioning from enabled to disabled, first disable
	if state.RollingRelease.Enabled.ValueBool() && !request.RollingRelease.Enabled {
		disabledRequest := client.UpdateRollingReleaseRequest{
			RollingRelease: client.RollingRelease{
				Enabled:         false,
				AdvancementType: "",
				Stages:          []client.RollingReleaseStage{},
			},
			ProjectID: request.ProjectID,
			TeamID:    request.TeamID,
		}

		_, err := r.client.UpdateRollingRelease(ctx, disabledRequest)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project rolling release",
				fmt.Sprintf("Could not disable project rolling release, unexpected error: %s",
					err,
				),
			)
			return
		}
	}

	// If we're transitioning from disabled to enabled, first create in disabled state
	if !state.RollingRelease.Enabled.ValueBool() && request.RollingRelease.Enabled {
		disabledRequest := client.UpdateRollingReleaseRequest{
			RollingRelease: client.RollingRelease{
				Enabled:         false,
				AdvancementType: "",
				Stages:          []client.RollingReleaseStage{},
			},
			ProjectID: request.ProjectID,
			TeamID:    request.TeamID,
		}

		_, err := r.client.UpdateRollingRelease(ctx, disabledRequest)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating project rolling release",
				fmt.Sprintf("Could not create project rolling release in disabled state, unexpected error: %s",
					err,
				),
			)
			return
		}
	}

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
		"enabled":          result.RollingRelease.Enabled.ValueBool(),
		"advancement_type": result.RollingRelease.AdvancementType.ValueString(),
		"stages":           result.RollingRelease.Stages,
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
