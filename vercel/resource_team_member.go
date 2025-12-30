package vercel

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &teamMemberResource{}
	_ resource.ResourceWithConfigure   = &teamMemberResource{}
	_ resource.ResourceWithImportState = &teamMemberResource{}
)

func newTeamMemberResource() resource.Resource {
	return &teamMemberResource{}
}

type teamMemberResource struct {
	client *client.Client
}

func (r *teamMemberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_member"
}

func (r *teamMemberResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *teamMemberResource) userIsUnconfirmed(ctx context.Context, req planmodifier.SetRequest, resp *setplanmodifier.RequiresReplaceIfFuncResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	var state TeamMember
	diags := req.Plan.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.RequiresReplace = !state.Confirmed.ValueBool()
}

func (r *teamMemberResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provider a resource for managing a team member.\n\n~> **Note:** Users can no longer be added to a team by their user_id. This field is maintained purely for backwards compatibility.",
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Description: "The ID of the existing Vercel Team.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseNonNullStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Description:        "The ID of the user to add to the team. Must specify one of user_id or email.",
				DeprecationMessage: "Users can no longer be added to a team by their user_id. This field is maintained purely for backwards compatibility.",
				Optional:           true,
				Computed:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseNonNullStateForUnknown(),
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("user_id"),
						path.MatchRoot("email"),
					),
				},
			},
			"email": schema.StringAttribute{
				Description: "The email of the user to add to the team. Must specify one of user_id or email.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseNonNullStateForUnknown(),
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("user_id"),
						path.MatchRoot("email"),
					),
				},
			},
			"role": schema.StringAttribute{
				Description: "The role that the user should have in the team. One of 'MEMBER', 'OWNER', 'VIEWER', 'VIEWER_FOR_PLUS', 'DEVELOPER', 'BILLING', 'SECURITY', or 'CONTRIBUTOR'. Depending on your Team's plan, some of these roles may be unavailable.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("MEMBER", "OWNER", "VIEWER", "VIEWER_FOR_PLUS", "DEVELOPER", "BILLING", "SECURITY", "CONTRIBUTOR"),
				},
			},
			"projects": schema.SetNestedAttribute{
				Description: "If access groups are enabled on the team, and the user is a CONTRIBUTOR, `projects`, `access_groups` or both must be specified. A set of projects that the user should be granted access to, along with their role in each project.",
				Optional:    true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplaceIf(r.userIsUnconfirmed, "Can only update projects on confirmed team members", "Can only update projects on confirmed team members"),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role": schema.StringAttribute{
							Description: "The role that the user should have in the project.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("ADMIN", "PROJECT_VIEWER", "PROJECT_DEVELOPER"),
							},
						},
						"project_id": schema.StringAttribute{
							Description: "The ID of the project that the user should be granted access to.",
							Required:    true,
						},
					},
				},
			},
			"access_groups": schema.SetAttribute{
				Description: "If access groups are enabled on the team, and the user is a CONTRIBUTOR, `projects`, `access_groups` or both must be specified. A set of access groups IDs that the user should be granted access to.",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			"confirmed": schema.BoolAttribute{
				Description: "Whether the user has confirmed their invitation.",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "The unique identifier for this resource. Format: team_id/email.",
				Computed:    true,
			},
		},
	}
}

type TeamMember struct {
	UserID       types.String `tfsdk:"user_id"`
	Email        types.String `tfsdk:"email"`
	TeamID       types.String `tfsdk:"team_id"`
	Role         types.String `tfsdk:"role"`
	Projects     types.Set    `tfsdk:"projects"`
	AccessGroups types.Set    `tfsdk:"access_groups"`
	Confirmed    types.Bool   `tfsdk:"confirmed"`
	ID           types.String `tfsdk:"id"`
}

func (t TeamMember) projects(ctx context.Context) ([]TeamMemberProject, diag.Diagnostics) {
	if t.Projects.IsNull() || t.Projects.IsUnknown() {
		return nil, nil
	}
	var tmps []TeamMemberProject
	diags := t.Projects.ElementsAs(ctx, &tmps, false)
	return tmps, diags
}

func (t TeamMember) accessGroups(ctx context.Context) ([]string, diag.Diagnostics) {
	if t.AccessGroups.IsNull() || t.AccessGroups.IsUnknown() {
		return nil, nil
	}
	var tmps []string
	diags := t.AccessGroups.ElementsAs(ctx, &tmps, false)
	return tmps, diags
}

type TeamMemberProject struct {
	Role      types.String `tfsdk:"role"`
	ProjectID types.String `tfsdk:"project_id"`
}

func (t TeamMember) toInviteTeamMemberRequest(ctx context.Context) (client.TeamMemberInviteRequest, diag.Diagnostics) {
	tmps, diags := t.projects(ctx)
	if diags.HasError() {
		return client.TeamMemberInviteRequest{}, diags
	}

	var projects []client.ProjectRole
	for _, p := range tmps {
		projects = append(projects, client.ProjectRole{
			ProjectID: p.ProjectID.ValueString(),
			Role:      p.Role.ValueString(),
		})
	}

	accessGroups, diags := t.accessGroups(ctx)
	if diags.HasError() {
		return client.TeamMemberInviteRequest{}, diags
	}

	return client.TeamMemberInviteRequest{
		TeamID:       t.TeamID.ValueString(),
		Email:        t.Email.ValueString(),
		Role:         t.Role.ValueString(),
		Projects:     projects,
		AccessGroups: accessGroups,
	}, diags
}

func areSameProjects(oldProjects, newProjects []TeamMemberProject) bool {
	if len(oldProjects) != len(newProjects) {
		return false
	}
	for _, p := range oldProjects {
		if !contains(newProjects, p) {
			return false
		}
	}
	for _, p := range newProjects {
		if !contains(oldProjects, p) {
			return false
		}
	}
	return true
}

func diffAccessGroups(oldAgs, newAgs []string) (toAdd, toRemove []string) {
	for _, n := range newAgs {
		if !contains(oldAgs, n) {
			toAdd = append(toAdd, n)
		}
	}
	for _, n := range oldAgs {
		if !contains(newAgs, n) {
			toRemove = append(toRemove, n)
		}
	}
	return
}

func (t TeamMember) toTeamMemberUpdateRequest(ctx context.Context, state TeamMember) (client.TeamMemberUpdateRequest, diag.Diagnostics) {
	tmps, diags := t.projects(ctx)
	if diags.HasError() {
		return client.TeamMemberUpdateRequest{}, diags
	}

	var projects []client.ProjectRole
	for _, p := range tmps {
		projects = append(projects, client.ProjectRole{
			ProjectID: p.ProjectID.ValueString(),
			Role:      p.Role.ValueString(),
		})
	}

	newAccessGroups, diags := t.accessGroups(ctx)
	if diags.HasError() {
		return client.TeamMemberUpdateRequest{}, diags
	}
	oldAccessGroups, diags := state.accessGroups(ctx)
	if diags.HasError() {
		return client.TeamMemberUpdateRequest{}, diags
	}

	toAdd, toRemove := diffAccessGroups(oldAccessGroups, newAccessGroups)
	return client.TeamMemberUpdateRequest{
		TeamID:               t.TeamID.ValueString(),
		UserID:               t.UserID.ValueString(),
		Role:                 t.Role.ValueString(),
		Projects:             projects,
		AccessGroupsToAdd:    toAdd,
		AccessGroupsToRemove: toRemove,
	}, nil
}

func (t TeamMember) toTeamMemberRemoveRequest() client.TeamMemberRemoveRequest {
	return client.TeamMemberRemoveRequest{
		UserID: t.UserID.ValueString(),
		TeamID: t.TeamID.ValueString(),
	}
}

var projectsElemType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"role":       types.StringType,
		"project_id": types.StringType,
	},
}

func convertResponseToTeamMember(response client.TeamMember, plan TeamMember) TeamMember {
	var projectsAttrs []attr.Value
	for _, p := range response.Projects {
		projectsAttrs = append(projectsAttrs, types.ObjectValueMust(
			map[string]attr.Type{
				"role":       types.StringType,
				"project_id": types.StringType,
			},
			map[string]attr.Value{
				"role":       types.StringValue(p.Role),
				"project_id": types.StringValue(p.ProjectID),
			},
		))
	}
	projects := types.SetValueMust(projectsElemType, projectsAttrs)
	if len(projectsAttrs) == 0 {
		projects = types.SetNull(projectsElemType)
	}

	var ags []attr.Value
	for _, ag := range response.AccessGroups {
		ags = append(ags, types.StringValue(ag.ID))
	}
	accessGroups := types.SetValueMust(types.StringType, ags)
	if len(ags) == 0 {
		accessGroups = types.SetNull(types.StringType)
	}

	teamMember := TeamMember{
		UserID:       types.StringValue(response.UserID),
		Email:        types.StringValue(response.Email),
		TeamID:       plan.TeamID,
		Role:         types.StringValue(response.Role),
		Projects:     projects,
		AccessGroups: accessGroups,
		Confirmed:    types.BoolValue(response.Confirmed),
		ID:           types.StringValue(plan.TeamID.ValueString() + "/" + response.Email),
	}

	if !response.Confirmed {
		// The API doesn't return the projects or access groups for unconfirmed members, so we have to
		// manually set these fields to whatever was in the plan.
		teamMember.Projects = types.SetNull(projectsElemType)
		if !plan.Projects.IsUnknown() && !plan.Projects.IsNull() {
			teamMember.Projects = plan.Projects
		}
		teamMember.AccessGroups = types.SetNull(types.StringType)
		if !plan.AccessGroups.IsUnknown() && !plan.AccessGroups.IsNull() {
			teamMember.AccessGroups = plan.AccessGroups
		}
	}
	return teamMember
}

func (r *teamMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TeamMember
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if user_id is provided and return an error
	if !plan.UserID.IsNull() && !plan.UserID.IsUnknown() {
		resp.Diagnostics.AddError(
			"user_id is no longer supported",
			"Users can no longer be added to a team by their user_id. This field is maintained purely for backwards compatibility.",
		)
		return
	}

	request, diags := plan.toInviteTeamMemberRequest(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	res, err := r.client.InviteTeamMember(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error inviting Team Member",
			"Could not invite Team Member, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Error(ctx, "Invited Team Member", map[string]any{
		"team_id": plan.TeamID.ValueString(),
		"user_id": plan.UserID.ValueString(),
	})

	planAGs, diags := plan.accessGroups(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	planProjects, diags := plan.projects(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	var response client.TeamMember
	getRetry := Retry{
		Base:     200 * time.Millisecond,
		Attempts: 7,
	}
	err = getRetry.Do(func(attempt int) (shouldRetry bool, err error) {
		response, err = r.client.GetTeamMember(ctx, client.GetTeamMemberRequest{
			TeamID: plan.TeamID.ValueString(),
			UserID: res.UserID,
		})
		if client.NotFound(err) {
			return true, err
		}
		if err != nil {
			return true, fmt.Errorf("unexpected error: %w", err)
		}
		teamMember := convertResponseToTeamMember(response, plan)
		if teamMember.Role != plan.Role {
			tflog.Error(ctx, "Role has not yet propagated", map[string]any{})
			return true, fmt.Errorf("role has not yet propagated")
		}
		respAGs, _ := teamMember.accessGroups(ctx)
		if new, old := diffAccessGroups(respAGs, planAGs); len(new) > 0 || len(old) > 0 {
			tflog.Error(ctx, "AGs have not yet propagated", map[string]any{})
			return true, fmt.Errorf("access groups have not yet propagated")
		}
		respProjects, _ := teamMember.projects(ctx)
		if !areSameProjects(respProjects, planProjects) {
			tflog.Error(ctx, "Projects have not yet propagated", map[string]any{})
			return true, fmt.Errorf("projects have not yet propagated")
		}
		return false, err
	})
	tflog.Error(ctx, "Read team member after inviting", map[string]any{
		"team_id": plan.TeamID.ValueString(),
		"user_id": plan.UserID.ValueString(),
		"err":     err,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Team Member",
			"Could not read Team Member, unexpected error: "+err.Error(),
		)
		return
	}
	teamMember := convertResponseToTeamMember(response, plan)
	diags = resp.State.Set(ctx, teamMember)
	resp.Diagnostics.Append(diags...)
}

func (r *teamMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TeamMember
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// sleep to allow time for the user to propagate in the Vercel system.
	if os.Getenv("TF_ACC") == "true" {
		time.Sleep(5 * time.Second)
	}

	response, err := r.client.GetTeamMember(ctx, client.GetTeamMemberRequest{
		TeamID: state.TeamID.ValueString(),
		UserID: state.UserID.ValueString(),
	})
	tflog.Info(ctx, "Read team member", map[string]any{
		"team_id":  state.TeamID.ValueString(),
		"user_id":  state.UserID.ValueString(),
		"err":      err,
		"response": response,
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Team Member",
			"Could not read Team Member, unexpected error: "+err.Error(),
		)
	}
	teamMember := convertResponseToTeamMember(response, state)
	diags = resp.State.Set(ctx, teamMember)
	resp.Diagnostics.Append(diags...)
}

func (r *teamMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TeamMember
	diags := req.Plan.Get(ctx, &plan)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var state TeamMember
	diags = req.State.Get(ctx, &state)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	tflog.Error(ctx, "updating team member", map[string]any{
		"state": state,
		"plan":  plan,
	})
	request, diags := plan.toTeamMemberUpdateRequest(ctx, state)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	err := r.client.UpdateTeamMember(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Team Member",
			"Could not update Team Member, unexpected error: "+err.Error(),
		)
		return
	}
	var teamMember TeamMember
	planAGs, diags := plan.accessGroups(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	planProjects, diags := plan.projects(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	getRetry := Retry{
		Base:     200 * time.Millisecond,
		Attempts: 7,
	}
	err = getRetry.Do(func(attempt int) (shouldRetry bool, err error) {
		response, err := r.client.GetTeamMember(ctx, client.GetTeamMemberRequest{
			TeamID: plan.TeamID.ValueString(),
			UserID: plan.UserID.ValueString(),
		})
		if client.NotFound(err) {
			return true, err
		}
		tflog.Error(ctx, "Fetched team member", map[string]any{
			"team_id": plan.TeamID.ValueString(),
			"user_id": plan.UserID.ValueString(),
			"role":    response.Role,
		})
		teamMember = convertResponseToTeamMember(response, plan)
		if teamMember.Role != plan.Role {
			tflog.Error(ctx, "Role has not yet propagated", map[string]any{})
			return true, fmt.Errorf("role has not yet propagated")
		}
		respAGs, _ := teamMember.accessGroups(ctx)
		if new, old := diffAccessGroups(respAGs, planAGs); len(new) > 0 || len(old) > 0 {
			tflog.Error(ctx, "AGs have not yet propagated", map[string]any{})
			return true, fmt.Errorf("access groups have not yet propagated")
		}
		respProjects, _ := teamMember.projects(ctx)
		if !areSameProjects(respProjects, planProjects) {
			tflog.Error(ctx, "Projects have not yet propagated", map[string]any{})
			return true, fmt.Errorf("projects have not yet propagated")
		}
		return false, err
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Team Member",
			"Could not read Team Member, unexpected error: "+err.Error(),
		)
	}
	tflog.Info(ctx, "updated Team member", map[string]any{
		"team_id": request.TeamID,
		"user_id": request.UserID,
	})
	diags = resp.State.Set(ctx, teamMember)
	resp.Diagnostics.Append(diags...)
}

func (r *teamMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TeamMember
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Removing team member", map[string]any{
		"team_id": state.TeamID,
		"user_id": state.UserID,
	})

	err := r.client.RemoveTeamMember(ctx, state.toTeamMemberRemoveRequest())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing Team Member",
			"Could not remove Team Member, unexpected error: "+err.Error(),
		)
	}

	resp.State.RemoveResource(ctx)
}

// ImportState implements resource.ResourceWithImportState.
func (r *teamMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, userID, ok := splitInto2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Team Member",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/user_id\"", req.ID),
		)
	}

	var response client.TeamMember
	getRetry := Retry{
		Base:     200 * time.Millisecond,
		Attempts: 5,
	}
	err := getRetry.Do(func(attempt int) (shouldRetry bool, err error) {
		response, err = r.client.GetTeamMember(ctx, client.GetTeamMemberRequest{
			TeamID: teamID,
			UserID: userID,
		})
		if client.NotFound(err) {
			return true, err
		}
		return false, err
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Team Member",
			"Could not read Team Member, unexpected error: "+err.Error(),
		)
	}
	teamMember := convertResponseToTeamMember(response, TeamMember{
		TeamID: types.StringValue(teamID),
	})
	diags := resp.State.Set(ctx, teamMember)
	resp.Diagnostics.Append(diags...)
}
