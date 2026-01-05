package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var (
	_ resource.Resource              = &projectMembersResource{}
	_ resource.ResourceWithConfigure = &projectMembersResource{}
)

func newProjectMembersResource() resource.Resource {
	return &projectMembersResource{}
}

type projectMembersResource struct {
	client *client.Client
}

func (r *projectMembersResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_members"
}

func (r *projectMembersResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *projectMembersResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Manages members and their roles for a Vercel Project.

~> Note that this resource does not manage the complete set of members for a project, only the members that
are explicitly configured here. This is deliberately done to allow granular additions.
This, however, means config drift will not be detected for members that are added or removed outside of terraform.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for this resource.",
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
				Description:   "The team ID to add the project to. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the existing Vercel Project.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseNonNullStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"members": schema.SetNestedAttribute{
				Description: "The set of members to manage for this project.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"user_id": schema.StringAttribute{
							Description: "The ID of the user to add to the project. Exactly one of `user_id`, `email`, or `username` must be specified.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseNonNullStateForUnknown(),
							},
							Validators: []validator.String{
								stringvalidator.ExactlyOneOf(
									path.MatchRelative().AtParent().AtName("user_id"),
									path.MatchRelative().AtParent().AtName("email"),
									path.MatchRelative().AtParent().AtName("username"),
								),
							},
						},
						"email": schema.StringAttribute{
							Description: "The email of the user to add to the project. Exactly one of `user_id`, `email`, or `username` must be specified.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseNonNullStateForUnknown(),
							},
							Validators: []validator.String{
								stringvalidator.ExactlyOneOf(
									path.MatchRelative().AtParent().AtName("user_id"),
									path.MatchRelative().AtParent().AtName("email"),
									path.MatchRelative().AtParent().AtName("username"),
								),
							},
						},
						"username": schema.StringAttribute{
							Description: "The username of the user to add to the project. Exactly one of `user_id`, `email`, or `username` must be specified.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseNonNullStateForUnknown(),
							},
							Validators: []validator.String{
								stringvalidator.ExactlyOneOf(
									path.MatchRelative().AtParent().AtName("user_id"),
									path.MatchRelative().AtParent().AtName("email"),
									path.MatchRelative().AtParent().AtName("username"),
								),
							},
						},
						"role": schema.StringAttribute{
							Description: "The role that the user should have in the project. One of 'ADMIN', 'PROJECT_DEVELOPER', or 'PROJECT_VIEWER'.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("ADMIN", "PROJECT_DEVELOPER", "PROJECT_VIEWER"),
							},
						},
					},
				},
			},
		},
	}
}

type ProjectMembersModel struct {
	ID        types.String `tfsdk:"id"`
	TeamID    types.String `tfsdk:"team_id"`
	ProjectID types.String `tfsdk:"project_id"`
	Members   types.Set    `tfsdk:"members"`
}

type ProjectMemberItem struct {
	UserID   types.String `tfsdk:"user_id"`
	Email    types.String `tfsdk:"email"`
	Username types.String `tfsdk:"username"`
	Role     types.String `tfsdk:"role"`
}

func (m ProjectMembersModel) members(ctx context.Context) ([]ProjectMemberItem, diag.Diagnostics) {
	var members []ProjectMemberItem
	diags := m.Members.ElementsAs(ctx, &members, false)
	return members, diags
}

var memberAttrType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"user_id":  types.StringType,
		"email":    types.StringType,
		"username": types.StringType,
		"role":     types.StringType,
	},
}

func (r *projectMembersResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectMembersModel
	diags := req.Plan.Get(ctx, &plan)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	planMembers, diags := plan.members(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	var requestMembers []client.ProjectMember
	for _, m := range planMembers {
		requestMembers = append(requestMembers, client.ProjectMember{
			UserID:   m.UserID.ValueString(),
			Username: m.Username.ValueString(),
			Email:    m.Email.ValueString(),
			Role:     m.Role.ValueString(),
		})
	}

	err := r.client.AddProjectMembers(ctx, client.AddProjectMembersRequest{
		ProjectID: plan.ProjectID.ValueString(),
		TeamID:    plan.TeamID.ValueString(),
		Members:   requestMembers,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error adding Project Members",
			fmt.Sprintf("Could not add Project Members, unexpected error: %s", err),
		)
		return
	}

	members, err := r.client.ListProjectMembers(ctx, client.GetProjectMembersRequest{
		TeamID:    plan.TeamID.ValueString(),
		ProjectID: plan.ProjectID.ValueString(),
	})
	tflog.Trace(ctx, "read project members", map[string]any{
		"team_id":    plan.TeamID.ValueString(),
		"project_id": plan.ProjectID.ValueString(),
		"members":    members,
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Project Members",
			fmt.Sprintf("Could not read Project Members, unexpected error: %s", err),
		)
		return
	}

	// Convert API response to model
	var memberItems []attr.Value
	for _, member := range members {
		if terraformHasMember(planMembers, member) {
			memberItems = append(memberItems, types.ObjectValueMust(memberAttrType.AttrTypes, map[string]attr.Value{
				"user_id":  types.StringValue(member.UserID),
				"email":    types.StringValue(member.Email),
				"username": types.StringValue(member.Username),
				"role":     types.StringValue(member.Role),
			}))
		}
	}

	plan.Members = types.SetValueMust(memberAttrType, memberItems)
	plan.TeamID = types.StringValue(r.client.TeamID(plan.TeamID.ValueString()))
	plan.ID = types.StringValue(plan.ProjectID.ValueString())
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

// diffMembers compares the state and planned members to determine which members need to be added, removed, or updated
func diffMembers(stateMembers, plannedMembers []ProjectMemberItem) (toAdd, toRemove, toUpdate []ProjectMemberItem) {
	stateMap := map[string]ProjectMemberItem{}
	plannedMap := map[string]ProjectMemberItem{}

	for _, member := range stateMembers {
		stateMap[member.UserID.ValueString()] = member
	}

	for _, member := range plannedMembers {
		stateMember, inState := stateMap[member.UserID.ValueString()]
		if member.UserID.IsUnknown() || member.Email.IsUnknown() || member.Username.IsUnknown() || !inState {
			// Then the member hasn't been created yet, so add it.
			toAdd = append(toAdd, member)
			continue
		}
		if _, ok := stateMap[member.UserID.ValueString()]; !ok {
			// Then the member hasn't been created yet, so add it.
			toAdd = append(toAdd, member)
			continue
		}

		// Add to planned, so we can reverse look up ones to remove later.
		plannedMap[member.UserID.ValueString()] = member
		if inState && stateMember.Role != member.Role {
			toUpdate = append(toUpdate, member)
		}
	}

	// Find members to remove (in state but not in plan)
	for key, member := range stateMap {
		if _, exists := plannedMap[key]; !exists {
			toRemove = append(toRemove, member)
		}
	}

	return toAdd, toRemove, toUpdate
}

func terraformHasMember(stateMembers []ProjectMemberItem, member client.ProjectMember) bool {
	for _, m := range stateMembers {
		if m.UserID.ValueString() == member.UserID || m.Email.ValueString() == member.Email || m.Username.ValueString() == member.Username {
			return true
		}
	}
	return false
}

func (r *projectMembersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectMembersModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stateMembers, diags := state.members(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	members, err := r.client.ListProjectMembers(ctx, client.GetProjectMembersRequest{
		TeamID:    state.TeamID.ValueString(),
		ProjectID: state.ProjectID.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Project Members",
			fmt.Sprintf("Could not read Project Members, unexpected error: %s", err),
		)
		return
	}

	// Convert API response to model
	var memberItems []attr.Value
	for _, member := range members {
		if terraformHasMember(stateMembers, member) {
			memberItems = append(memberItems, types.ObjectValueMust(memberAttrType.AttrTypes, map[string]attr.Value{
				"user_id":  types.StringValue(member.UserID),
				"email":    types.StringValue(member.Email),
				"username": types.StringValue(member.Username),
				"role":     types.StringValue(member.Role),
			}))
		}
	}

	state.Members = types.SetValueMust(memberAttrType, memberItems)
	state.TeamID = types.StringValue(r.client.TeamID(state.TeamID.ValueString()))
	state.ID = types.StringValue(state.ProjectID.ValueString())
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *projectMembersResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ProjectMembersModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planMembers, diags := plan.members(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Get current members
	currentMembers, err := r.client.ListProjectMembers(ctx, client.GetProjectMembersRequest{
		ProjectID: plan.ProjectID.ValueString(),
		TeamID:    plan.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading current Project Members",
			fmt.Sprintf("Could not read current Project Members: %s", err),
		)
		return
	}

	// Create a map of current members for easy lookup
	currentMemberMap := make(map[string]client.ProjectMember)
	for _, member := range currentMembers {
		currentMemberMap[member.UserID] = member
	}

	// Process planned members
	plannedMembers, diags := plan.members(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	stateMembers, diags := state.members(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	toAdd, toRemove, toUpdate := diffMembers(stateMembers, plannedMembers)
	tflog.Info(ctx, "update project members", map[string]any{
		"toAdd":    toAdd,
		"toRemove": toRemove,
		"toUpdate": toUpdate,
	})

	// Remove members that are no longer in the plan
	var remove []string
	for _, r := range toRemove {
		remove = append(remove, r.UserID.ValueString())
	}

	if len(remove) > 0 {
		err = r.client.RemoveProjectMembers(ctx, client.RemoveProjectMembersRequest{
			ProjectID: plan.ProjectID.ValueString(),
			TeamID:    plan.TeamID.ValueString(),
			Members:   remove,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error removing Project Members",
				fmt.Sprintf("Could not remove Project Members: %s", err),
			)
			return
		}
	}

	var add []client.ProjectMember
	for _, a := range toAdd {
		add = append(add, client.ProjectMember{
			UserID:   a.UserID.ValueString(),
			Username: a.Username.ValueString(),
			Email:    a.Email.ValueString(),
			Role:     a.Role.ValueString(),
		})
	}
	if len(add) > 0 {
		err = r.client.AddProjectMembers(ctx, client.AddProjectMembersRequest{
			ProjectID: plan.ProjectID.ValueString(),
			TeamID:    plan.TeamID.ValueString(),
			Members:   add,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error adding Project Members",
				fmt.Sprintf("Could not add Project Members: %s", err),
			)
			return
		}
	}

	var update []client.UpdateProjectMemberRequest
	for _, u := range toUpdate {
		update = append(update, client.UpdateProjectMemberRequest{
			UserID: u.UserID.ValueString(),
			Role:   u.Role.ValueString(),
		})
	}
	if len(update) > 0 {
		err = r.client.UpdateProjectMembers(ctx, client.UpdateProjectMembersRequest{
			ProjectID: plan.ProjectID.ValueString(),
			TeamID:    plan.TeamID.ValueString(),
			Members:   update,
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating Project Members",
				fmt.Sprintf("Could not update Project Members: %s", err),
			)
			return
		}
	}

	members, err := r.client.ListProjectMembers(ctx, client.GetProjectMembersRequest{
		TeamID:    state.TeamID.ValueString(),
		ProjectID: state.ProjectID.ValueString(),
	})
	tflog.Info(ctx, "read project members", map[string]any{
		"team_id":    state.TeamID.ValueString(),
		"project_id": state.ProjectID.ValueString(),
		"members":    members,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Project Members",
			fmt.Sprintf("Could not read Project Members, unexpected error: %s", err),
		)
		return
	}

	// Convert API response to model
	var memberItems []attr.Value
	for _, member := range members {
		if terraformHasMember(planMembers, member) {
			memberItems = append(memberItems, types.ObjectValueMust(memberAttrType.AttrTypes, map[string]attr.Value{
				"user_id":  types.StringValue(member.UserID),
				"email":    types.StringValue(member.Email),
				"username": types.StringValue(member.Username),
				"role":     types.StringValue(member.Role),
			}))
		}
	}

	state.Members = types.SetValueMust(memberAttrType, memberItems)
	state.TeamID = types.StringValue(r.client.TeamID(state.TeamID.ValueString()))
	state.ID = types.StringValue(state.ProjectID.ValueString())
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *projectMembersResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectMembersModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	members, diags := state.members(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var remove []string
	for _, m := range members {
		remove = append(remove, m.UserID.ValueString())
	}

	err := r.client.RemoveProjectMembers(ctx, client.RemoveProjectMembersRequest{
		ProjectID: state.ProjectID.ValueString(),
		TeamID:    state.TeamID.ValueString(),
		Members:   remove,
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing Project Members",
			fmt.Sprintf("Could not remove Project Members: %s", err),
		)
		return
	}

}
