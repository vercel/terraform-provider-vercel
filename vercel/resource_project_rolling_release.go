package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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

// Custom validator for advancement_type
type advancementTypeValidator struct{}

func (v advancementTypeValidator) Description(ctx context.Context) string {
	return "advancement_type must be either 'automatic' or 'manual-approval'"
}

func (v advancementTypeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v advancementTypeValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if value != "automatic" && value != "manual-approval" {
		resp.Diagnostics.AddError(
			"Invalid advancement_type",
			fmt.Sprintf("advancement_type must be either 'automatic' or 'manual-approval', got: %s", value),
		)
	}
}

// Schema returns the schema information for a project rolling release resource.
func (r *projectRollingReleaseResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages rolling release configuration for a Vercel project.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project.",
				Required:           true,
			},
			"team_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the team the project exists in.",
				Required:           true,
			},
			"rolling_release": schema.SingleNestedAttribute{
				MarkdownDescription: "The rolling release configuration.",
				Required:           true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether rolling releases are enabled.",
						Required:           true,
					},
					"advancement_type": schema.StringAttribute{
						MarkdownDescription: "The type of advancement between stages. Must be either 'automatic' or 'manual-approval'. Required when enabled is true.",
						Optional:           true,
						Computed:           true,
						Validators: []validator.String{
							advancementTypeValidator{},
						},
					},
					"stages": schema.ListNestedAttribute{
						MarkdownDescription: "The stages of the rolling release. Required when enabled is true.",
						Optional:           true,
						Computed:           true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"target_percentage": schema.Int64Attribute{
									MarkdownDescription: "The percentage of traffic to route to this stage.",
									Required:           true,
								},
								"duration": schema.Int64Attribute{
									MarkdownDescription: "The duration in minutes to wait before advancing to the next stage. Required for all stages except the final stage when using automatic advancement.",
									Optional:           true,
									Computed:           true,
								},
								"require_approval": schema.BoolAttribute{
									MarkdownDescription: "Whether approval is required before advancing to the next stage.",
									Optional:           true,
									Computed:           true,
								},
							},
						},
					},
				},
			},
		},
	}
}

type TFRollingReleaseStage struct {
	TargetPercentage types.Int64  `tfsdk:"target_percentage"`
	Duration        types.Int64  `tfsdk:"duration"`
	RequireApproval types.Bool   `tfsdk:"require_approval"`
}

// TFRollingRelease reflects the state terraform stores internally for a project rolling release.
type TFRollingRelease struct {
	Enabled         types.Bool          `tfsdk:"enabled"`
	AdvancementType types.String       `tfsdk:"advancement_type"`
	Stages          types.List         `tfsdk:"stages"`
}

// ProjectRollingRelease reflects the state terraform stores internally for a project rolling release.
type TFRollingReleaseInfo struct {
	RollingRelease TFRollingRelease `tfsdk:"rolling_release"`
	ProjectID      types.String     `tfsdk:"project_id"`
	TeamID         types.String     `tfsdk:"team_id"`
}

type RollingReleaseStage struct {
	TargetPercentage int  `json:"targetPercentage"`
	Duration        *int `json:"duration,omitempty"`
	RequireApproval bool `json:"requireApproval"`
}

type RollingRelease struct {
	Enabled         bool                `json:"enabled"`
	AdvancementType string             `json:"advancementType"`
	Stages          []RollingReleaseStage `json:"stages"`
}

type UpdateRollingReleaseRequest struct {
	RollingRelease RollingRelease `json:"rollingRelease"`
	ProjectID      string         `json:"-"`
	TeamID         string         `json:"-"`
}

func (e *TFRollingReleaseInfo) toUpdateRollingReleaseRequest() (client.UpdateRollingReleaseRequest, diag.Diagnostics) {
	var stages []client.RollingReleaseStage
	var advancementType string
	var diags diag.Diagnostics

	if e.RollingRelease.Enabled.ValueBool() {
		if !e.RollingRelease.AdvancementType.IsNull() {
			advancementType = e.RollingRelease.AdvancementType.ValueString()
		} else {
			advancementType = "manual-approval" // Default to manual-approval if not specified
		}

		// Convert stages from types.List to []client.RollingReleaseStage
		var tfStages []TFRollingReleaseStage
		if !e.RollingRelease.Stages.IsNull() && !e.RollingRelease.Stages.IsUnknown() {
			diags = e.RollingRelease.Stages.ElementsAs(context.Background(), &tfStages, false)
			if diags.HasError() {
				return client.UpdateRollingReleaseRequest{}, diags
			}
			stages = make([]client.RollingReleaseStage, len(tfStages))
			for i, stage := range tfStages {
				// For automatic advancement, set a default duration if not provided
				if advancementType == "automatic" {
					var duration int = 60 // Default duration in minutes
					if !stage.Duration.IsNull() {
						duration = int(stage.Duration.ValueInt64())
					}
					stages[i] = client.RollingReleaseStage{
						TargetPercentage: int(stage.TargetPercentage.ValueInt64()),
						Duration:         &duration,
						RequireApproval:  stage.RequireApproval.ValueBool(),
					}
				} else {
					// For manual approval, omit duration field completely
					stages[i] = client.RollingReleaseStage{
						TargetPercentage: int(stage.TargetPercentage.ValueInt64()),
						RequireApproval:  stage.RequireApproval.ValueBool(),
					}
				}
			}
		}
	} else {
		// When disabled, don't send any stages to the API
		stages = []client.RollingReleaseStage{}
	}

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

func convertStages(stages []client.RollingReleaseStage, advancementType string, planStages []TFRollingReleaseStage, enabled bool, ctx context.Context) (types.List, diag.Diagnostics) {
	// If disabled, always return plan stages to preserve state
	if !enabled && len(planStages) > 0 {
		elements := make([]attr.Value, len(planStages))
		for i, stage := range planStages {
			// For disabled state, ensure duration is known
			var duration types.Int64
			if stage.Duration.IsUnknown() {
				duration = types.Int64Null()
			} else {
				duration = stage.Duration
			}

			elements[i] = types.ObjectValueMust(
				map[string]attr.Type{
					"target_percentage": types.Int64Type,
					"duration":         types.Int64Type,
					"require_approval": types.BoolType,
				},
				map[string]attr.Value{
					"target_percentage": stage.TargetPercentage,
					"duration":         duration,
					"require_approval": stage.RequireApproval,
				},
			)
		}
		return types.ListValueFrom(ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"target_percentage": types.Int64Type,
				"duration":         types.Int64Type,
				"require_approval": types.BoolType,
			},
		}, elements)
	}

	// If no stages from API and no plan stages, return empty list
	if len(stages) == 0 {
		return types.ListValueFrom(ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"target_percentage": types.Int64Type,
				"duration":         types.Int64Type,
				"require_approval": types.BoolType,
			},
		}, []attr.Value{})
	}

	elements := make([]attr.Value, len(stages))
	for i, stage := range stages {
		targetPercentage := types.Int64Value(int64(stage.TargetPercentage))
		requireApproval := types.BoolValue(stage.RequireApproval)
		var duration types.Int64

		// If we have plan stages, preserve the values but ensure they're known
		if i < len(planStages) {
			targetPercentage = planStages[i].TargetPercentage
			requireApproval = planStages[i].RequireApproval
			
			// Handle duration based on advancement type
			if advancementType == "automatic" {
				if planStages[i].Duration.IsUnknown() {
					// For unknown values, use API value or default
					if stage.Duration != nil {
						duration = types.Int64Value(int64(*stage.Duration))
					} else {
						duration = types.Int64Value(60) // Default duration in minutes
					}
				} else {
					duration = planStages[i].Duration
				}
			} else {
				duration = types.Int64Null() // Manual approval doesn't use duration
			}
		} else {
			// Only set duration for automatic advancement
			if advancementType == "automatic" {
				if stage.Duration != nil {
					duration = types.Int64Value(int64(*stage.Duration))
				} else {
					duration = types.Int64Value(60) // Default duration in minutes
				}
			} else {
				duration = types.Int64Null()
			}
		}

		elements[i] = types.ObjectValueMust(
			map[string]attr.Type{
				"target_percentage": types.Int64Type,
				"duration":         types.Int64Type,
				"require_approval": types.BoolType,
			},
			map[string]attr.Value{
				"target_percentage": targetPercentage,
				"duration":         duration,
				"require_approval": requireApproval,
			},
		)
	}

	return types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"target_percentage": types.Int64Type,
			"duration":         types.Int64Type,
			"require_approval": types.BoolType,
		},
	}, elements)
}

func convertResponseToTFRollingRelease(response client.RollingReleaseInfo, plan *TFRollingReleaseInfo, ctx context.Context) (TFRollingReleaseInfo, diag.Diagnostics) {
	var diags diag.Diagnostics

	result := TFRollingReleaseInfo{
		RollingRelease: TFRollingRelease{
			Enabled: types.BoolValue(response.RollingRelease.Enabled),
		},
		ProjectID: types.StringValue(response.ProjectID),
		TeamID:    types.StringValue(response.TeamID),
	}

	// Get plan stages if available
	var planStages []TFRollingReleaseStage
	if plan != nil && !plan.RollingRelease.Stages.IsNull() && !plan.RollingRelease.Stages.IsUnknown() {
		diags.Append(plan.RollingRelease.Stages.ElementsAs(ctx, &planStages, false)...)
		if diags.HasError() {
			return result, diags
		}
	}

	if response.RollingRelease.Enabled {
		result.RollingRelease.AdvancementType = types.StringValue(response.RollingRelease.AdvancementType)
	} else {
		result.RollingRelease.AdvancementType = types.StringNull()
	}

	// Convert stages, passing enabled state to ensure proper preservation
	stages, stagesDiags := convertStages(
		response.RollingRelease.Stages,
		response.RollingRelease.AdvancementType,
		planStages,
		response.RollingRelease.Enabled,
		ctx,
	)
	diags.Append(stagesDiags...)
	if diags.HasError() {
		return result, diags
	}
	result.RollingRelease.Stages = stages

	return result, diags
}

// Create will create a new rolling release config on a Vercel project.
// This is called automatically by the provider when a new resource should be created.
func (r *projectRollingReleaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Starting rolling release creation")

	var plan TFRollingReleaseInfo
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Got plan from request", map[string]any{
		"project_id": plan.ProjectID.ValueString(),
		"team_id": plan.TeamID.ValueString(),
		"enabled": plan.RollingRelease.Enabled.ValueBool(),
		"advancement_type": plan.RollingRelease.AdvancementType.ValueString(),
		"stages": plan.RollingRelease.Stages,
	})

	_, err := r.client.GetProject(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString())
	if client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Error creating project rolling release",
			"Could not find project, please make sure both the project_id and team_id match the project and team you wish to deploy to.",
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project rolling release",
			"Error reading project information, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Project exists, creating rolling release")

	updateRequest, diags := plan.toUpdateRollingReleaseRequest()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.client.UpdateRollingRelease(ctx, updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project rolling release",
			"Could not create project rolling release, unexpected error: "+err.Error(),
		)
		return
	}

	result, diags := convertResponseToTFRollingRelease(response, &plan, ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Log the values for debugging
	tflog.Debug(ctx, "created project rolling release", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
		"enabled": result.RollingRelease.Enabled.ValueBool(),
		"advancement_type": result.RollingRelease.AdvancementType.ValueString(),
		"stages": result.RollingRelease.Stages,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read an rolling release of a Vercel project by requesting it from the Vercel API, and will update terraform
// with this information.
func (r *projectRollingReleaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TFRollingReleaseInfo
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
			fmt.Sprintf("Could not get project rolling release %s %s, unexpected error: %s",
				err,
			),
		)
		return
	}

	result, diags := convertResponseToTFRollingRelease(out, nil, ctx)
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

// Delete deletes a Vercel project rolling release.
func (r *projectRollingReleaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TFRollingReleaseInfo
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteRollingRelease(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project rolling release",
			fmt.Sprintf(
				"Could not delete project rolling release %s, unexpected error: %s",
				state.ProjectID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted project rolling release", map[string]any{
		"team_id":    state.TeamID.ValueString(),
		"project_id": state.ProjectID.ValueString(),
	})
}

// Update updates the project rolling release of a Vercel project state.
func (r *projectRollingReleaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TFRollingReleaseInfo
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest, diags := plan.toUpdateRollingReleaseRequest()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.client.UpdateRollingRelease(ctx, updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project rolling release",
			"Could not update project rolling release, unexpected error: "+err.Error(),
		)
		return
	}

	result, diags := convertResponseToTFRollingRelease(response, &plan, ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Log the values for debugging
	tflog.Debug(ctx, "updated project rolling release", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
		"enabled": result.RollingRelease.Enabled.ValueBool(),
		"advancement_type": result.RollingRelease.AdvancementType.ValueString(),
		"stages": result.RollingRelease.Stages,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
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

	result, diags := convertResponseToTFRollingRelease(out, nil, ctx)
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
