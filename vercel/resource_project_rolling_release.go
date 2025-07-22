package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

// durationValidator validates that duration is only present when advancement_type is "automatic"
type durationValidator struct{}

func (v durationValidator) Description(ctx context.Context) string {
	return "duration can only be set when advancement_type is 'automatic'"
}

func (v durationValidator) MarkdownDescription(ctx context.Context) string {
	return "`duration` can only be set when `advancement_type` is `automatic`"
}

func (v durationValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	// Get the parent advancement_type value
	parentPath := req.Path.ParentPath()
	advancementTypePath := parentPath.AtName("advancement_type")

	var advancementType types.String
	diags := req.Config.GetAttribute(ctx, advancementTypePath, &advancementType)
	if diags.HasError() {
		return
	}

	// Check if duration is set
	durationAttr, exists := req.ConfigValue.Attributes()["duration"]
	if !exists {
		return
	}

	duration := durationAttr.(types.Int64)
	if duration.IsNull() || duration.IsUnknown() {
		if advancementType.ValueString() == "manual-approval" {
			return
		}
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("duration"),
			"Invalid duration configuration",
			"duration can only be set when advancement_type is 'automatic'",
		)
	}

	// If duration is set but advancement_type is not "automatic", add an error
	if advancementType.ValueString() != "automatic" {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("duration"),
			"Invalid duration configuration",
			"duration must be set when advancement_type is 'automatic'",
		)
	}
}

// Schema returns the schema information for a project rolling release resource.
func (r *projectRollingReleaseResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Resource for a Vercel project rolling release configuration.",
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
			"advancement_type": schema.StringAttribute{
				MarkdownDescription: "The type of advancement for the rolling release. Must be either 'automatic' or 'manual-approval'.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("automatic", "manual-approval"),
				},
			},
			"stages": schema.ListNestedAttribute{
				MarkdownDescription: "The stages for the rolling release configuration.",
				Required:            true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.SizeAtMost(10),
				},
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
							MarkdownDescription: "The duration in minutes to wait before advancing to the next stage. Required for automatic advancement type.",
							Optional:            true,
							Validators: []validator.Int64{
								int64validator.Between(1, 10000),
							},
						},
					},
					Validators: []validator.Object{
						durationValidator{},
					},
				},
			},
		},
	}
}

// ProjectRollingRelease reflects the state terraform stores internally for a project rolling release.
type RollingReleaseInfo struct {
	AdvancementType types.String `tfsdk:"advancement_type"`
	Stages          types.List   `tfsdk:"stages"`
	ProjectID       types.String `tfsdk:"project_id"`
	TeamID          types.String `tfsdk:"team_id"`
}

func (e *RollingReleaseInfo) toCreateRollingReleaseRequest() (client.CreateRollingReleaseRequest, diag.Diagnostics) {
	var stages []client.RollingReleaseStage
	var diags diag.Diagnostics

	advancementType := e.AdvancementType.ValueString()

	// Convert stages using a more robust approach
	var rollingReleaseStages []RollingReleaseStage
	diags = e.Stages.ElementsAs(context.Background(), &rollingReleaseStages, false)

	// Add all stages from config
	stages = make([]client.RollingReleaseStage, len(rollingReleaseStages))
	for i, stage := range rollingReleaseStages {
		clientStage := client.RollingReleaseStage{
			TargetPercentage: int(stage.TargetPercentage.ValueInt64()),
			RequireApproval:  advancementType == "manual-approval",
		}

		// Add duration for automatic advancement type
		if advancementType == "automatic" && !stage.Duration.IsNull() && !stage.Duration.IsUnknown() {
			duration := int(stage.Duration.ValueInt64())
			clientStage.Duration = &duration
		}
		if advancementType == "automatic" && (stage.Duration.IsNull() || stage.Duration.IsUnknown()) {
			duration := int(60)
			clientStage.Duration = &duration
		}

		stages[i] = clientStage
	}

	// Add terminal stage (100%) without approval
	stages = append(stages, client.RollingReleaseStage{
		TargetPercentage: 100,
		RequireApproval:  false,
	})

	// Log the request for debugging
	tflog.Info(context.Background(), "converting to update request", map[string]any{
		"advancement_type": advancementType,
		"stages_count":     len(stages),
	})

	return client.CreateRollingReleaseRequest{
		RollingRelease: client.RollingRelease{
			Enabled:         true,
			AdvancementType: advancementType,
			Stages:          stages,
		},
		ProjectID: e.ProjectID.ValueString(),
		TeamID:    e.TeamID.ValueString(),
	}, diags
}

func (e *RollingReleaseInfo) toUpdateRollingReleaseRequest() (client.UpdateRollingReleaseRequest, diag.Diagnostics) {
	var stages []client.RollingReleaseStage
	var diags diag.Diagnostics

	advancementType := e.AdvancementType.ValueString()

	// Convert stages using a more robust approach
	var rollingReleaseStages []RollingReleaseStage
	diags = e.Stages.ElementsAs(context.Background(), &rollingReleaseStages, false)

	// Add all stages from config
	stages = make([]client.RollingReleaseStage, len(rollingReleaseStages))
	for i, stage := range rollingReleaseStages {
		clientStage := client.RollingReleaseStage{
			TargetPercentage: int(stage.TargetPercentage.ValueInt64()),
			RequireApproval:  advancementType == "manual-approval",
		}

		// Add duration for automatic advancement type
		if advancementType == "automatic" && !stage.Duration.IsNull() && !stage.Duration.IsUnknown() {
			duration := int(stage.Duration.ValueInt64())
			clientStage.Duration = &duration
		}

		if advancementType == "automatic" && (stage.Duration.IsNull() || stage.Duration.IsUnknown()) {
			duration := int(60)
			clientStage.Duration = &duration
		}

		stages[i] = clientStage
	}

	// Add terminal stage (100%) without approval
	stages = append(stages, client.RollingReleaseStage{
		TargetPercentage: 100,
		RequireApproval:  false,
	})

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

	// If disabled or advancementType is empty, check if we have stages to determine if it's configured
	if !response.RollingRelease.Enabled || response.RollingRelease.AdvancementType == "" {
		// If the API response shows disabled or advancementType is empty, but the plan has configuration, use the plan's values
		if plan != nil &&
			!plan.AdvancementType.IsNull() && plan.AdvancementType.ValueString() != "" &&
			!plan.Stages.IsNull() && len(plan.Stages.Elements()) > 0 {

			result.AdvancementType = plan.AdvancementType
			result.Stages = plan.Stages
			return result, diags
		}

		// For import or when no plan is available, check if there are stages in the response
		// If there are stages, assume the rolling release is configured and use the response data
		if len(response.RollingRelease.Stages) > 0 {
			// Try to infer the advancement type from the stages
			advancementType := "manual-approval" // Default to manual-approval
			for _, stage := range response.RollingRelease.Stages {
				if stage.Duration != nil {
					advancementType = "automatic"
					break
				}
			}
			result.AdvancementType = types.StringValue(advancementType)

			// Convert the stages from the response
			var rollingReleaseStages []RollingReleaseStage
			for _, stage := range response.RollingRelease.Stages {
				// Skip the terminal stage (100%)
				if stage.TargetPercentage == 100 {
					continue
				}

				rollingReleaseStage := RollingReleaseStage{
					TargetPercentage: types.Int64Value(int64(stage.TargetPercentage)),
				}

				// Add duration if it exists (for automatic advancement type)
				if stage.Duration != nil {
					rollingReleaseStage.Duration = types.Int64Value(int64(*stage.Duration))
				}

				rollingReleaseStages = append(rollingReleaseStages, rollingReleaseStage)
			}

			// Convert to Terraform types
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
		} else {
			result.AdvancementType = types.StringNull()
			result.Stages = types.ListNull(RollingReleaseStageElementType)
		}
		return result, diags
	}

	// Set the advancement type
	result.AdvancementType = types.StringValue(response.RollingRelease.AdvancementType)

	// Convert API stages to stages (excluding terminal stage)
	var rollingReleaseStages []RollingReleaseStage
	for _, stage := range response.RollingRelease.Stages {
		// Skip the terminal stage (100%)
		if stage.TargetPercentage == 100 {
			continue
		}

		rollingReleaseStage := RollingReleaseStage{
			TargetPercentage: types.Int64Value(int64(stage.TargetPercentage)),
		}

		// Add duration if it exists (for automatic advancement type)
		if stage.Duration != nil {
			rollingReleaseStage.Duration = types.Int64Value(int64(*stage.Duration))
		}

		rollingReleaseStages = append(rollingReleaseStages, rollingReleaseStage)
	}

	// Convert to Terraform types
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

	// First, check if a rolling release already exists
	existingRelease, err := r.client.GetRollingRelease(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString())
	if err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Error checking existing project rolling release",
			fmt.Sprintf("Could not check if project rolling release exists, unexpected error: %s",
				err,
			),
		)
		return
	}

	// If a rolling release already exists and is enabled, return an error
	if err == nil && existingRelease.RollingRelease.Enabled {
		resp.Diagnostics.AddError(
			"Project rolling release already exists",
			fmt.Sprintf("A rolling release is already configured for project %s. Please use the update operation instead.",
				plan.ProjectID.ValueString(),
			),
		)
		return
	}

	// Convert plan to client request
	request, diags := plan.toCreateRollingReleaseRequest()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateRollingRelease(ctx, request)
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

	err := r.client.DeleteRollingRelease(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString())
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
