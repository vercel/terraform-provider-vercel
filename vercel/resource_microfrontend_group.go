package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v2/client"
)

var (
	_ resource.Resource              = &microfrontendGroupResource{}
	_ resource.ResourceWithConfigure = &microfrontendGroupResource{}
)

func newMicrofrontendGroupResource() resource.Resource {
	return &microfrontendGroupResource{}
}

type microfrontendGroupResource struct {
	client *client.Client
}

func (r *microfrontendGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_microfrontend_group"
}

func (r *microfrontendGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Schema returns the schema information for a microfrontendGroup resource.
func (r *microfrontendGroupResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Microfrontend Group resource.

A Microfrontend Group is a definition of a microfrontend belonging to a Vercel Team. 

Example:

resource "vercel_microfrontend_group" "my-microfrontend-group" {
  name = "microfrontend test"
  projects = {
    (vercel_project.my-parent-project.id) = {
      is_default_app = true
    }
    (vercel_project.my-child-project.id) = {
      is_default_app = false
    }
  }
}
`,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "A human readable name for the microfrontends group.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description:   "A unique identifier for the group of microfrontends. Example: mfe_12HKQaOmR5t5Uy6vdcQsNIiZgHGB",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"slug": schema.StringAttribute{
				Description: "A slugified version of the name.",
				Computed:    true,
			},
			"team_id": schema.StringAttribute{
				Description:   "The team ID to add the microfrontend group to. Required when configuring a team resource if a default team has not been set in the provider.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"projects": schema.MapNestedAttribute{
				Description: "A map of project ids to project configuration that belong to the microfrontend group.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"is_default_app": schema.BoolAttribute{
							Description:   "Whether the project is the default app for the microfrontend group. Microfrontend groups must have exactly one default app. (Omit false values)",
							Optional:      true,
							Computed:      true,
							PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown(), boolplanmodifier.RequiresReplace()},
							Validators: []validator.Bool{
								boolvalidator.ExactlyOneOf(
									path.MatchRoot("projects").AtAnyMapKey().AtName("is_default_app"),
								),
							},
						},
						"default_route": schema.StringAttribute{
							Description:   "The default route for the project. Used for the screenshot of deployments.",
							Optional:      true,
							Computed:      true,
							PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						},
						"route_observability_to_this_project": schema.BoolAttribute{
							Description:   "Whether the project is route observability for this project. If dalse, the project will be route observability for all projects to the default project.",
							Optional:      true,
							Computed:      true,
							Default:       booldefault.StaticBool(true),
							PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
						},
					},
				},
				Validators:    []validator.Map{mapvalidator.SizeAtLeast(1)},
				PlanModifiers: []planmodifier.Map{mapplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

type MicrofrontendProject struct {
	IsDefaultApp                    types.Bool   `tfsdk:"is_default_app"`
	DefaultRoute                    types.String `tfsdk:"default_route"`
	RouteObservabilityToThisProject types.Bool   `tfsdk:"route_observability_to_this_project"`
}

type MicrofrontendGroup struct {
	TeamID   types.String                    `tfsdk:"team_id"`
	ID       types.String                    `tfsdk:"id"`
	Name     types.String                    `tfsdk:"name"`
	Slug     types.String                    `tfsdk:"slug"`
	Projects map[string]MicrofrontendProject `tfsdk:"projects"`
}

func convertResponseToMicrofrontendGroup(group client.MicrofrontendGroup, projects map[string]client.MicrofrontendProject) MicrofrontendGroup {
	projectResponse := map[string]MicrofrontendProject{}
	for projectID, p := range projects {
		projectResponse[projectID] = MicrofrontendProject{
			IsDefaultApp:                    types.BoolValue(p.IsDefaultApp),
			DefaultRoute:                    types.StringValue(p.DefaultRoute),
			RouteObservabilityToThisProject: types.BoolValue(p.RouteObservabilityToThisProject),
		}
	}
	return MicrofrontendGroup{
		ID:       types.StringValue(group.ID),
		Name:     types.StringValue(group.Name),
		Slug:     types.StringValue(group.Slug),
		TeamID:   types.StringValue(group.TeamID),
		Projects: projectResponse,
	}
}

func (r *microfrontendGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MicrofrontendGroup
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError(
			"Error getting microfrontendGroup plan",
			"Error getting microfrontendGroup plan",
		)
		return
	}

	tflog.Info(ctx, "creating microfrontend group", map[string]interface{}{
		"team_id": plan.TeamID.ValueString(),
		"name":    plan.Name.ValueString(),
		"plan":    plan,
	})

	cdr := client.MicrofrontendGroup{
		Name:   plan.Name.ValueString(),
		TeamID: plan.TeamID.ValueString(),
	}

	groupResponse, err := r.client.CreateMicrofrontendGroup(ctx, cdr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating microfrontend group",
			"Could not create microfrontend group, unexpected error: "+err.Error(),
		)
		return
	}

	projectResponse := map[string]client.MicrofrontendProject{}
	for projectID, project := range plan.Projects {
		p, err := r.client.AddOrUpdateMicrofrontendProject(ctx, client.MicrofrontendProject{
			IsDefaultApp:                    project.IsDefaultApp.ValueBool(),
			DefaultRoute:                    project.DefaultRoute.ValueString(),
			RouteObservabilityToThisProject: project.RouteObservabilityToThisProject.ValueBool(),
			MicrofrontendGroupID:            groupResponse.ID,
			TeamID:                          plan.TeamID.ValueString(),
			ProjectID:                       projectID,
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating microfrontend project",
				"Could not create microfrontend project, unexpected error: "+err.Error(),
			)
			return
		}
		projectResponse[projectID] = p
	}

	result := convertResponseToMicrofrontendGroup(groupResponse, projectResponse)
	tflog.Info(ctx, "created microfrontend group", map[string]interface{}{
		"team_id":  result.TeamID.ValueString(),
		"group_id": result.ID.ValueString(),
		"slug":     result.Slug.ValueString(),
		"name":     result.Name.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *microfrontendGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MicrofrontendGroup
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetMicrofrontendGroup(ctx, state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading microfrontend group",
			fmt.Sprintf("Could not get microfrontend group %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToMicrofrontendGroup(out, out.Projects)
	tflog.Info(ctx, "read microfrontend group", map[string]interface{}{
		"team_id":  result.TeamID.ValueString(),
		"group_id": result.ID.ValueString(),
		"slug":     result.Slug.ValueString(),
		"name":     result.Name.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *microfrontendGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MicrofrontendGroup
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError(
			"Error getting microfrontend group plan",
			"Error getting microfrontend group plan",
		)
		return
	}

	var state MicrofrontendGroup
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	for projectID, project := range state.Projects {
		_, exists := plan.Projects[projectID]
		if !exists {
			tflog.Info(ctx, "removing microfrontend project", map[string]interface{}{
				"project_id": projectID,
			})
			_, err := r.client.RemoveMicrofrontendProject(ctx, client.MicrofrontendProject{
				ProjectID:                       projectID,
				IsDefaultApp:                    project.IsDefaultApp.ValueBool(),
				DefaultRoute:                    project.DefaultRoute.ValueString(),
				RouteObservabilityToThisProject: project.RouteObservabilityToThisProject.ValueBool(),
				MicrofrontendGroupID:            state.ID.ValueString(),
				TeamID:                          state.TeamID.ValueString(),
			})
			if err != nil {
				resp.Diagnostics.AddError(
					"Error removing microfrontend project "+projectID,
					"Could not remove microfrontend project, unexpected error: "+err.Error(),
				)
				return
			}
		}
	}

	projects := map[string]client.MicrofrontendProject{}
	for projectID, project := range plan.Projects {
		tflog.Info(ctx, "adding / updating microfrontend project", map[string]interface{}{
			"project_id": projectID,
		})
		updatedProject, err := r.client.AddOrUpdateMicrofrontendProject(ctx, client.MicrofrontendProject{
			ProjectID:                       projectID,
			IsDefaultApp:                    project.IsDefaultApp.ValueBool(),
			DefaultRoute:                    project.DefaultRoute.ValueString(),
			RouteObservabilityToThisProject: project.RouteObservabilityToThisProject.ValueBool(),
			MicrofrontendGroupID:            state.ID.ValueString(),
			TeamID:                          state.TeamID.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error adding microfrontend project "+projectID,
				"Could not add microfrontend project, unexpected error: "+err.Error(),
			)
			return
		}
		projects[projectID] = updatedProject
	}

	out, err := r.client.UpdateMicrofrontendGroup(ctx, client.MicrofrontendGroup{
		ID:     state.ID.ValueString(),
		Name:   plan.Name.ValueString(),
		TeamID: state.TeamID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating microfrontend group",
			fmt.Sprintf(
				"Could not update microfrontend group %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "updated microfrontend group", map[string]interface{}{
		"team_id":  out.TeamID,
		"group_id": out.ID,
		"name":     out.Name,
		"slug":     out.Slug,
	})

	result := convertResponseToMicrofrontendGroup(out, projects)

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *microfrontendGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MicrofrontendGroup
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	for projectID, project := range state.Projects {
		tflog.Info(ctx, "removing microfrontend project", map[string]interface{}{
			"project_id": projectID,
		})
		_, err := r.client.RemoveMicrofrontendProject(ctx, client.MicrofrontendProject{
			ProjectID:                       projectID,
			IsDefaultApp:                    project.IsDefaultApp.ValueBool(),
			DefaultRoute:                    project.DefaultRoute.ValueString(),
			RouteObservabilityToThisProject: project.RouteObservabilityToThisProject.ValueBool(),
			MicrofrontendGroupID:            state.ID.ValueString(),
			TeamID:                          state.TeamID.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error removing microfrontend project "+projectID,
				"Could not remove microfrontend project, unexpected error: "+err.Error(),
			)
			return
		}
	}
	_, err := r.client.DeleteMicrofrontendGroup(ctx, client.MicrofrontendGroup{
		ID:     state.ID.ValueString(),
		TeamID: state.TeamID.ValueString(),
		Slug:   state.Slug.ValueString(),
		Name:   state.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting microfrontendGroup",
			fmt.Sprintf(
				"Could not delete microfrontendGroup %s, unexpected error: %s",
				state.ID.ValueString(),
				err,
			),
		)
		return
	}
	tflog.Info(ctx, "deleted microfrontendGroup", map[string]any{
		"group_id": state.ID.ValueString(),
	})
}
