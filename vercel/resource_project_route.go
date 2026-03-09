package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var (
	_ resource.Resource                = &projectRouteResource{}
	_ resource.ResourceWithConfigure   = &projectRouteResource{}
	_ resource.ResourceWithImportState = &projectRouteResource{}
)

func newProjectRouteResource() resource.Resource {
	return &projectRouteResource{}
}

type projectRouteResource struct {
	client *client.Client
}

func (r *projectRouteResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_route"
}

func (r *projectRouteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *projectRouteResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Project Route resource.

This resource manages one live project-level routing rule for a Vercel project. Each mutation stages a new routing-rules version and promotes it immediately so Terraform state reflects production traffic behavior.

Reads and imports intentionally target the live version and ignore unpublished staged drafts created outside Terraform.

Position is applied when the rule is created or replaced. Use before and after with reference_route_id when you need deterministic ordering across multiple Terraform-managed rules.

The Vercel API does not return placement metadata for an individual rule. Terraform preserves the configured position on normal reads, but imported routes start with no position in state.

~> This resource refuses to mutate a project while it has an unpublished staged routing-rules version. Publish, restore, or discard the draft first.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The Vercel route ID.",
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the project to manage a routing rule for.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the team the project exists under. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "A human-readable name for the rule.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 256),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "An optional description of the rule.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 1024),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the rule is enabled.",
			},
			"src_syntax": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The source pattern syntax. You can usually omit this and let Vercel infer it from `route.src`.",
				Validators: []validator.String{
					stringvalidator.OneOf("equals", "path-to-regexp", "regex"),
				},
			},
			"route_type": schema.StringAttribute{
				Computed:    true,
				Description: "The computed route type returned by Vercel. One of `rewrite`, `redirect`, `set_status`, or `transform`.",
			},
			"position": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Where to insert the rule when it is created or replaced. This metadata is not returned by the API, so imported routes do not infer it.",
				Validators: []validator.Object{
					projectRoutePositionValidator{},
				},
				Attributes: map[string]schema.Attribute{
					"placement": schema.StringAttribute{
						Required:    true,
						Description: "Where to place the rule. One of `start`, `end`, `before`, or `after`.",
						Validators: []validator.String{
							stringvalidator.OneOf("start", "end", "before", "after"),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"reference_route_id": schema.StringAttribute{
						Optional:    true,
						Description: "The existing route ID to place this rule before or after.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplaceIfConfigured(),
						},
					},
				},
			},
			"route": schema.SingleNestedAttribute{
				Required:    true,
				Description: "The routing rule definition.",
				Validators: []validator.Object{
					projectRouteDefinitionValidator{},
				},
				Attributes: map[string]schema.Attribute{
					"src": schema.StringAttribute{
						Required:    true,
						Description: "The source pattern to match.",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"dest": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The destination for rewrites or redirects.",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"headers": schema.MapAttribute{
						Optional:    true,
						Computed:    true,
						ElementType: types.StringType,
						Description: "Headers to set for the matched request.",
					},
					"case_sensitive": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Whether the `src` matcher is case-sensitive.",
					},
					"status": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "The HTTP status code to set for redirects or status-only rules.",
						Validators: []validator.Int64{
							int64validator.Between(100, 999),
						},
					},
					"has": schema.ListNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Conditions that must be present for the rule to match.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									Required:    true,
									Description: "The condition type. One of `host`, `header`, `cookie`, or `query`.",
									Validators: []validator.String{
										stringvalidator.OneOf("host", "header", "cookie", "query"),
									},
								},
								"key": schema.StringAttribute{
									Optional:    true,
									Computed:    true,
									Description: "The key to match for `header`, `cookie`, or `query` conditions.",
								},
								"value": schema.StringAttribute{
									Optional:    true,
									Computed:    true,
									Description: "The value to match.",
								},
							},
							Validators: []validator.Object{
								projectRouteConditionValidator{},
							},
						},
					},
					"missing": schema.ListNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Conditions that must be absent for the rule to match.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									Required:    true,
									Description: "The condition type. One of `host`, `header`, `cookie`, or `query`.",
									Validators: []validator.String{
										stringvalidator.OneOf("host", "header", "cookie", "query"),
									},
								},
								"key": schema.StringAttribute{
									Optional:    true,
									Computed:    true,
									Description: "The key to match for `header`, `cookie`, or `query` conditions.",
								},
								"value": schema.StringAttribute{
									Optional:    true,
									Computed:    true,
									Description: "The value to match.",
								},
							},
							Validators: []validator.Object{
								projectRouteConditionValidator{},
							},
						},
					},
					"transforms": schema.ListNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Transforms applied to the request or response when the rule matches.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									Required:    true,
									Description: "The transform target. One of `request.headers`, `request.query`, or `response.headers`.",
									Validators: []validator.String{
										stringvalidator.OneOf("request.headers", "request.query", "response.headers"),
									},
								},
								"op": schema.StringAttribute{
									Required:    true,
									Description: "The transform operation. One of `append`, `set`, or `delete`.",
									Validators: []validator.String{
										stringvalidator.OneOf("append", "set", "delete"),
									},
								},
								"target": schema.StringAttribute{
									Optional:    true,
									Computed:    true,
									Description: "A JSON document describing the transform target. Prefer `jsonencode(...)` when setting this.",
									Validators: []validator.String{
										validateJSON(),
									},
								},
								"args": schema.StringAttribute{
									Optional:    true,
									Computed:    true,
									Description: "A JSON document containing transform arguments. Prefer `jsonencode(...)` when setting this.",
									Validators: []validator.String{
										validateJSON(),
									},
								},
								"env": schema.ListAttribute{
									Optional:    true,
									Computed:    true,
									ElementType: types.StringType,
									Description: "Environment names that gate this transform.",
								},
							},
						},
					},
					"respect_origin_cache_control": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Whether the rule should respect cache control headers from the origin.",
					},
				},
			},
		},
	}
}

type projectRouteConditionValidator struct{}

func (v projectRouteConditionValidator) Description(ctx context.Context) string {
	return "Condition keys are only used for header, cookie, and query matches"
}

func (v projectRouteConditionValidator) MarkdownDescription(ctx context.Context) string {
	return "Condition keys are only used for `header`, `cookie`, and `query` matches"
}

func (v projectRouteConditionValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	conditionType := req.ConfigValue.Attributes()["type"].(types.String)
	key := req.ConfigValue.Attributes()["key"].(types.String)

	if conditionType.IsNull() || conditionType.IsUnknown() {
		return
	}

	switch conditionType.ValueString() {
	case "host":
		if !key.IsNull() && !key.IsUnknown() && key.ValueString() != "" {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("key"),
				"Invalid condition key",
				"`key` cannot be set when `type` is `host`.",
			)
		}
	default:
		if key.IsNull() || key.IsUnknown() || key.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("key"),
				"Missing condition key",
				"`key` must be set when `type` is `header`, `cookie`, or `query`.",
			)
		}
	}
}

type projectRouteDefinitionValidator struct{}

func (v projectRouteDefinitionValidator) Description(ctx context.Context) string {
	return "A route must perform at least one action"
}

func (v projectRouteDefinitionValidator) MarkdownDescription(ctx context.Context) string {
	return "A route must perform at least one action"
}

func (v projectRouteDefinitionValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	dest := req.ConfigValue.Attributes()["dest"].(types.String)
	status := req.ConfigValue.Attributes()["status"].(types.Int64)
	headers := req.ConfigValue.Attributes()["headers"].(types.Map)
	transforms := req.ConfigValue.Attributes()["transforms"].(types.List)

	if !dest.IsNull() && !dest.IsUnknown() && dest.ValueString() != "" {
		return
	}
	if !status.IsNull() && !status.IsUnknown() {
		return
	}
	if !headers.IsNull() && !headers.IsUnknown() && len(headers.Elements()) > 0 {
		return
	}
	if !transforms.IsNull() && !transforms.IsUnknown() && len(transforms.Elements()) > 0 {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Missing route action",
		"Set at least one of `dest`, `status`, `headers`, or `transforms` so the rule changes request handling.",
	)
}

type projectRoutePositionValidator struct{}

func (v projectRoutePositionValidator) Description(ctx context.Context) string {
	return "before/after positions require a reference route ID"
}

func (v projectRoutePositionValidator) MarkdownDescription(ctx context.Context) string {
	return "before/after positions require a reference route ID"
}

func (v projectRoutePositionValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	placement := req.ConfigValue.Attributes()["placement"].(types.String)
	referenceRouteID := req.ConfigValue.Attributes()["reference_route_id"].(types.String)

	if placement.IsNull() || placement.IsUnknown() {
		return
	}

	switch placement.ValueString() {
	case "before", "after":
		if referenceRouteID.IsUnknown() {
			return
		}
		if referenceRouteID.IsNull() || referenceRouteID.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("reference_route_id"),
				"Missing reference route ID",
				"`reference_route_id` must be set when `placement` is `before` or `after`.",
			)
		}
	default:
		if referenceRouteID.IsUnknown() {
			return
		}
		if !referenceRouteID.IsNull() && referenceRouteID.ValueString() != "" {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("reference_route_id"),
				"Unexpected reference route ID",
				"`reference_route_id` can only be set when `placement` is `before` or `after`.",
			)
		}
	}
}

func (r *projectRouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectRouteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.GetProject(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString())
	if client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Error creating project route",
			"Could not find project, please make sure both the project_id and team_id match the project and team you wish to configure.",
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project route",
			"Error reading project information, unexpected error: "+err.Error(),
		)
		return
	}

	unlock := projectRouteLocks.Lock(projectRoutesResourceID(r.client.TeamID(plan.TeamID.ValueString()), plan.ProjectID.ValueString()))
	defer unlock()

	if err := r.ensureNoStagedProjectRoutes(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error creating project route", err.Error())
		return
	}

	routeInput, diags := plan.projectRoute().toClientInput(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.AddProjectRoute(ctx, client.AddProjectRouteRequest{
		TeamID:    plan.TeamID.ValueString(),
		ProjectID: plan.ProjectID.ValueString(),
		Route:     routeInput,
		Position:  plan.Position.toClientPosition(),
	})
	if client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Error creating project route",
			"Could not find project, please make sure both the project_id and team_id match the project and team you wish to configure.",
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project route",
			"Could not create project route, unexpected error: "+err.Error(),
		)
		return
	}

	if err := r.promoteProjectRouteVersion(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString(), result.Version.ID); err != nil {
		resp.Diagnostics.AddError(
			"Error creating project route",
			"Could not promote project route, unexpected error: "+err.Error(),
		)
		return
	}

	state, diags, err := readProjectRoute(ctx, r.client, result.Route.ID, plan.ProjectID.ValueString(), plan.TeamID.ValueString(), plan.projectRoute(), plan.Position)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project route",
			fmt.Sprintf("Could not read project route %s %s %s, unexpected error: %s", plan.TeamID.ValueString(), plan.ProjectID.ValueString(), result.Route.ID, err),
		)
		return
	}

	tflog.Info(ctx, "created project route", map[string]any{
		"team_id":    state.TeamID.ValueString(),
		"project_id": state.ProjectID.ValueString(),
		"route_id":   state.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *projectRouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectRouteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, diags, err := readProjectRoute(ctx, r.client, state.ID.ValueString(), state.ProjectID.ValueString(), state.TeamID.ValueString(), state.projectRoute(), state.Position)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project route",
			fmt.Sprintf("Could not get project route %s %s %s, unexpected error: %s", state.TeamID.ValueString(), state.ProjectID.ValueString(), state.ID.ValueString(), err),
		)
		return
	}

	tflog.Info(ctx, "read project route", map[string]any{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
		"route_id":   result.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *projectRouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProjectRouteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ProjectRouteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !projectRoutePositionsEqual(plan.Position, state.Position) {
		resp.Diagnostics.AddError(
			"Error updating project route",
			"Changing `position` requires replacing the route resource.",
		)
		return
	}

	unlock := projectRouteLocks.Lock(projectRoutesResourceID(r.client.TeamID(plan.TeamID.ValueString()), plan.ProjectID.ValueString()))
	defer unlock()

	if err := r.ensureNoStagedProjectRoutes(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error updating project route", err.Error())
		return
	}

	routeInput, diags := plan.projectRoute().toClientInput(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.EditProjectRoute(ctx, client.EditProjectRouteRequest{
		TeamID:    plan.TeamID.ValueString(),
		ProjectID: plan.ProjectID.ValueString(),
		RouteID:   state.ID.ValueString(),
		Route:     &routeInput,
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project route",
			fmt.Sprintf("Could not update project route %s %s %s, unexpected error: %s", plan.TeamID.ValueString(), plan.ProjectID.ValueString(), state.ID.ValueString(), err),
		)
		return
	}

	if err := r.promoteProjectRouteVersion(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString(), result.Version.ID); err != nil {
		resp.Diagnostics.AddError(
			"Error updating project route",
			"Could not promote project route, unexpected error: "+err.Error(),
		)
		return
	}

	newState, diags, err := readProjectRoute(ctx, r.client, state.ID.ValueString(), plan.ProjectID.ValueString(), plan.TeamID.ValueString(), plan.projectRoute(), plan.Position)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project route",
			fmt.Sprintf("Could not read project route %s %s %s, unexpected error: %s", plan.TeamID.ValueString(), plan.ProjectID.ValueString(), state.ID.ValueString(), err),
		)
		return
	}

	tflog.Info(ctx, "updated project route", map[string]any{
		"team_id":    newState.TeamID.ValueString(),
		"project_id": newState.ProjectID.ValueString(),
		"route_id":   newState.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *projectRouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectRouteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	unlock := projectRouteLocks.Lock(projectRoutesResourceID(r.client.TeamID(state.TeamID.ValueString()), state.ProjectID.ValueString()))
	defer unlock()

	if err := r.ensureNoStagedProjectRoutes(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting project route", err.Error())
		return
	}

	result, err := r.client.DeleteProjectRoutes(ctx, client.DeleteProjectRoutesRequest{
		TeamID:    state.TeamID.ValueString(),
		ProjectID: state.ProjectID.ValueString(),
		RouteIDs:  []string{state.ID.ValueString()},
	})
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project route",
			fmt.Sprintf("Could not delete project route %s %s %s, unexpected error: %s", state.TeamID.ValueString(), state.ProjectID.ValueString(), state.ID.ValueString(), err),
		)
		return
	}

	if err := r.promoteProjectRouteVersion(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString(), result.Version.ID); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project route",
			"Could not promote deleted project route version, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "deleted project route", map[string]any{
		"team_id":    state.TeamID.ValueString(),
		"project_id": state.ProjectID.ValueString(),
		"route_id":   state.ID.ValueString(),
	})
}

func (r *projectRouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, routeID, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing project route",
			fmt.Sprintf("Invalid id %q specified. Expected \"team_id/project_id/route_id\" or \"project_id/route_id\".", req.ID),
		)
		return
	}

	result, diags, err := readProjectRoute(ctx, r.client, routeID, projectID, teamID, ProjectRoute{}, ProjectRoutePosition{})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Error importing project route",
			fmt.Sprintf("Could not find project route %s %s %s.", teamID, projectID, routeID),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing project route",
			fmt.Sprintf("Could not get project route %s %s %s, unexpected error: %s", teamID, projectID, routeID, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *projectRouteResource) ensureNoStagedProjectRoutes(ctx context.Context, projectID, teamID string) error {
	versions, err := r.client.GetProjectRouteVersions(ctx, projectID, teamID)
	if err != nil {
		return err
	}

	for _, version := range versions {
		if version.IsStaging {
			return fmt.Errorf("project %s has an unpublished staged routing-rules version (%s). Publish, restore, or discard it before managing `vercel_project_route` resources", projectID, version.ID)
		}
	}

	return nil
}

func (r *projectRouteResource) promoteProjectRouteVersion(ctx context.Context, projectID, teamID, versionID string) error {
	_, err := r.client.UpdateProjectRoutingRuleVersion(ctx, client.UpdateProjectRoutingRuleVersionRequest{
		TeamID:    teamID,
		ProjectID: projectID,
		ID:        versionID,
		Action:    "promote",
	})
	return err
}

func projectRoutePositionsEqual(a, b ProjectRoutePosition) bool {
	return a.Placement.Equal(b.Placement) && a.ReferenceRouteID.Equal(b.ReferenceRouteID)
}
