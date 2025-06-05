package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
func (r *projectRollingReleaseResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Project Rolling release resource.

A Project Rolling release resource defines an Rolling release on a Vercel Project.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/rolling-releases).
`,
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Description: "Whether the rolling release is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					// RequiresReplace is used to ensure that if the value changes, the resource will be replaced.
					boolplanmodifier.RequiresReplace(),
				},
			},
			"advancement_type": schema.StringAttribute{
				Description: "The advancement type of the rolling release. Can be 'automatic' or 'manual-approve'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("manual-approve"),

				PlanModifiers: []planmodifier.String{
					// RequiresReplace is used to ensure that if the value changes, the resource will be replaced.
					stringplanmodifier.RequiresReplace(),
				},
			},
			"canary_response_header": schema.BoolAttribute{
				Description: "Whether the canary response header is enabled. This header is used to identify canary deployments.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					// RequiresReplace is used to ensure that if the value changes, the resource will be replaced.
					boolplanmodifier.RequiresReplace(),
				},
			},
			"stages": schema.ListAttribute{
				Description: "A list of stages for the rolling release. Each stage has a target percentage and duration.",
				Optional:    true,
				Computed:    true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"targetPercentage": types.Float64Type,
						"duration":         types.Float64Type,
					},
				},
				PlanModifiers: []planmodifier.List{
					// RequiresReplace is used to ensure that if the value changes, the resource will be replaced.
					listplanmodifier.RequiresReplace(),
				},
			},

			"project_id": schema.StringAttribute{
				Description:   "The ID of the Project for the retention policy",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the Vercel team.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

type TFRollingReleaseStage struct {
	TargetPercentage float64 `tfsdk:"targetPercentage,omitempty"`
	Duration         float64 `tfsdk:"duration,omitempty"`
	RequireApproval  bool    `tfsdk:"requireApproval,omitempty"`
}

// TFRollingRelease reflects the state terraform stores internally for a project rolling release.
type TFRollingRelease struct {
	Enabled              bool                    `tfsdk:"enabled,omitempty"`
	AdvancementType      string                  `tfsdk:"advancementType,omitempty"`
	CanaryResponseHeader bool                    `tfsdk:"canaryResponseHeader,omitempty"`
	Stages               []TFRollingReleaseStage `tfsdk:"stages,omitempty"`
}

// ProjectRollingRelease reflects the state terraform stores internally for a project rolling release.
type TFRollingReleaseInfo struct {
	RollingRelease TFRollingRelease `tfsdk:"rollingRelease,omitempty"`
	ProjectID      string           `tfsdk:"project_id"`
	TeamID         string           `tfsdk:"team_id"`
}

func (e *TFRollingReleaseInfo) toUpdateRollingReleaseRequest() client.UpdateRollingReleaseRequest {
	return client.UpdateRollingReleaseRequest{
		RollingRelease: client.RollingRelease{
			Enabled:              e.RollingRelease.Enabled,
			AdvancementType:      e.RollingRelease.AdvancementType,
			CanaryResponseHeader: e.RollingRelease.CanaryResponseHeader,
			Stages:               make([]client.RollingReleaseStage, len(e.RollingRelease.Stages)),
		},
		ProjectID: e.ProjectID,
		TeamID:    e.TeamID,
	}
}

func convertStages(stages []client.RollingReleaseStage) []TFRollingReleaseStage {
	result := make([]TFRollingReleaseStage, len(stages))
	for i, stage := range stages {
		result[i] = TFRollingReleaseStage{
			TargetPercentage: stage.TargetPercentage,
			Duration:         stage.Duration,
			RequireApproval:  stage.RequireApproval,
		}
	}
	return result
}

// convertResponseToTFRollingRelease is used to populate terraform state based on an API response.
// Where possible, values from the API response are used to populate state. If not possible,
// values from plan are used.
func convertResponseToTFRollingRelease(response client.RollingReleaseInfo) TFRollingReleaseInfo {
	return TFRollingReleaseInfo{
		RollingRelease: TFRollingRelease{
			Enabled:              response.RollingRelease.Enabled,
			AdvancementType:      response.RollingRelease.AdvancementType,
			CanaryResponseHeader: response.RollingRelease.CanaryResponseHeader,
			Stages:               convertStages(response.RollingRelease.Stages),
		},
		ProjectID: response.ProjectID,
		TeamID:    response.TeamID,
	}
}

// Create will create a new rolling release config on a Vercel project.
// This is called automatically by the provider when a new resource should be created.
func (r *projectRollingReleaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TFRollingReleaseInfo
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.GetProject(ctx, plan.ProjectID, plan.TeamID)
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

	response, err := r.client.UpdateRollingRelease(ctx, plan.toUpdateRollingReleaseRequest())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project rolling release",
			"Could not create project rolling release, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToTFRollingRelease(response)

	tflog.Info(ctx, "created project rolling release", map[string]any{
		"team_id":    result.TeamID,
		"project_id": result.ProjectID,
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

	out, err := r.client.GetRollingRelease(ctx, state.ProjectID, state.TeamID)
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

	result := convertResponseToTFRollingRelease(out)
	tflog.Info(ctx, "read project rolling release", map[string]any{
		"team_id":    result.TeamID,
		"project_id": result.ProjectID,
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

	err := r.client.DeleteRollingRelease(ctx, state.ProjectID, state.TeamID)
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project rolling release",
			fmt.Sprintf(
				"Could not delete project rolling release %s, unexpected error: %s",
				state.ProjectID,
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted project rolling release", map[string]any{
		"team_id":    state.TeamID,
		"project_id": state.ProjectID,
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

	response, err := r.client.UpdateRollingRelease(ctx, plan.toUpdateRollingReleaseRequest())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project rolling release",
			"Could not update project rolling release, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToTFRollingRelease(response)

	tflog.Info(ctx, "updated project rolling release", map[string]any{
		"team_id":    result.TeamID,
		"project_id": result.ProjectID,
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

	result := convertResponseToTFRollingRelease(out)
	tflog.Info(ctx, "imported project rolling release", map[string]any{
		"team_id":    result.TeamID,
		"project_id": result.ProjectID,
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
