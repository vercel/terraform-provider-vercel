package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v2/client"
)

var (
	_ resource.Resource               = &projectEnvironmentVariablesResource{}
	_ resource.ResourceWithConfigure  = &projectEnvironmentVariablesResource{}
	_ resource.ResourceWithModifyPlan = &projectEnvironmentVariablesResource{}
)

func newProjectEnvironmentVariablesResource() resource.Resource {
	return &projectEnvironmentVariablesResource{}
}

type projectEnvironmentVariablesResource struct {
	client *client.Client
}

func (r *projectEnvironmentVariablesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_environment_variables"
}

func (r *projectEnvironmentVariablesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Schema returns the schema information for a project environment variable resource.
func (r *projectEnvironmentVariablesResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a resource for managing a number of Project Environment Variables.

This resource defines multiple Environment Variables on a Vercel Project.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/concepts/projects/environment-variables).

~> Terraform currently provides this Project Environment Variables resource (multiple Environment Variables), a single Project Environment Variable Resource, and a Project resource with Environment Variables defined in-line via the ` + "`environment` field" + `.
At this time you cannot use a Vercel Project resource with in-line ` + "`environment` in conjunction with any `vercel_project_environment_variables` or `vercel_project_environment_variable`" + ` resources. Doing so will cause a conflict of settings and will overwrite Environment Variables.
`,
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the Vercel project.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the Vercel team. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"variables": schema.SetNestedAttribute{
				Required:    true,
				Description: "A set of Environment Variables that should be configured for the project.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the Environment Variable.",
							Computed:    true,
						},
						"key": schema.StringAttribute{
							Required:      true,
							PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
							Description:   "The name of the Environment Variable.",
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "The value of the Environment Variable.",
							// Sensitive:   true,
						},
						"target": schema.SetAttribute{
							Required:    true,
							Description: "The environments that the Environment Variable should be present on. Valid targets are either `production`, `preview`, or `development`.",
							ElementType: types.StringType,
							Validators: []validator.Set{
								setvalidator.ValueStringsAre(stringvalidator.OneOf("production", "preview", "development")),
								setvalidator.SizeAtLeast(1),
							},
						},
						"git_branch": schema.StringAttribute{
							Optional:    true,
							Description: "The git branch of the Environment Variable.",
						},
						"sensitive": schema.BoolAttribute{
							Description:   "Whether the Environment Variable is sensitive or not.",
							Optional:      true,
							Computed:      true,
							PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
						},
						"comment": schema.StringAttribute{
							Description: "A comment explaining what the environment variable is for.",
							Optional:    true,
							Computed:    true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(0, 1000),
							},
						},
					},
				},
			},
		},
	}
}

// ProjectEnvironmentVariables reflects the state terraform stores internally for project environment variables.
type ProjectEnvironmentVariables struct {
	TeamID    types.String `tfsdk:"team_id"`
	ProjectID types.String `tfsdk:"project_id"`
	Variables types.Set    `tfsdk:"variables"`
}

func (p *ProjectEnvironmentVariables) environment(ctx context.Context) ([]EnvironmentItem, error) {
	if p.Variables.IsNull() {
		return nil, nil
	}

	var vars []EnvironmentItem
	err := p.Variables.ElementsAs(ctx, &vars, true)
	if err != nil {
		return nil, fmt.Errorf("error reading project environment variables: %s", err)
	}
	return vars, nil
}

func (r *projectEnvironmentVariablesResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var config ProjectEnvironmentVariables
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	environment, err := config.environment(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project environment variables",
			"Could not read environment variables, unexpected error: "+err.Error(),
		)
		return
	}

	// Should be at least one variable
	if len(environment) == 0 {
		return
	}

	// work out if there are any new env vars that are specifying sensitive = false
	var nonSensitiveEnvVars []path.Path
	for i, e := range environment {
		if e.ID.ValueString() != "" {
			continue
		}
		if e.Sensitive.IsUnknown() || e.Sensitive.IsNull() || e.Sensitive.ValueBool() {
			continue
		}
		nonSensitiveEnvVars = append(
			nonSensitiveEnvVars,
			path.Root("variables").
				AtSetValue(config.Variables.Elements()[i]).
				AtName("sensitive"),
		)
	}

	if len(nonSensitiveEnvVars) == 0 {
		return
	}

	// if sensitive is explicitly set to `false`, then validate that an env var can be created with the given
	// team sensitive environment variable policy.
	team, err := r.client.Team(ctx, config.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error validating project environment variables",
			"Could not validate project environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	if team.SensitiveEnvironmentVariablePolicy == nil || *team.SensitiveEnvironmentVariablePolicy != "on" {
		// the policy isn't enabled
		return
	}

	for _, p := range nonSensitiveEnvVars {
		resp.Diagnostics.AddAttributeError(
			p,
			"Project Environment Variables Invalid",
			"This team has a policy that forces all environment variables to be sensitive. Please remove the `sensitive` field for your environment variables or set the `sensitive` field to `true` in your configuration.",
		)
	}
}

func (e *ProjectEnvironmentVariables) toCreateEnvironmentVariablesRequest(ctx context.Context) (r client.CreateEnvironmentVariablesRequest, err error) {
	envs, err := e.environment(ctx)
	if err != nil {
		return r, err
	}

	variables := []client.EnvironmentVariableRequest{}
	for _, e := range envs {
		target := []string{}
		for _, t := range e.Target {
			target = append(target, t.ValueString())
		}
		var envVariableType string
		if e.Sensitive.ValueBool() {
			envVariableType = "sensitive"
		} else {
			envVariableType = "encrypted"
		}
		variables = append(variables, client.EnvironmentVariableRequest{
			Key:       e.Key.ValueString(),
			Value:     e.Value.ValueString(),
			Target:    target,
			Type:      envVariableType,
			GitBranch: e.GitBranch.ValueStringPointer(),
			Comment:   e.Comment.ValueString(),
		})
	}

	return client.CreateEnvironmentVariablesRequest{
		ProjectID:            e.ProjectID.ValueString(),
		TeamID:               e.TeamID.ValueString(),
		EnvironmentVariables: variables,
	}, nil
}

// convertResponseToProjectEnvironmentVariables is used to populate terraform state based on an API response.
// Where possible, values from the API response are used to populate state. If not possible,
// values from plan are used.
func convertResponseToProjectEnvironmentVariables(ctx context.Context, response []client.EnvironmentVariable, plan ProjectEnvironmentVariables) (ProjectEnvironmentVariables, error) {
	environment, err := plan.environment(ctx)
	if err != nil {
		return ProjectEnvironmentVariables{}, fmt.Errorf("error reading project environment variables: %s", err)
	}

	var env []attr.Value
	for _, e := range response {
		target := []attr.Value{}
		for _, t := range e.Target {
			target = append(target, types.StringValue(t))
		}
		value := types.StringValue(e.Value)
		if e.Type == "sensitive" {
			value = types.StringNull()
		}
		if !e.Decrypted || e.Type == "sensitive" {
			for _, p := range environment {
				if p.Key.ValueString() == e.Key && hasSameTarget(p, e.Target) {
					value = p.Value
					break
				}
			}
		}

		env = append(env, types.ObjectValueMust(
			map[string]attr.Type{
				"key":   types.StringType,
				"value": types.StringType,
				"target": types.SetType{
					ElemType: types.StringType,
				},
				"git_branch": types.StringType,
				"id":         types.StringType,
				"sensitive":  types.BoolType,
				"comment":    types.StringType,
			},
			map[string]attr.Value{
				"key":        types.StringValue(e.Key),
				"value":      value,
				"target":     types.SetValueMust(types.StringType, target),
				"git_branch": types.StringPointerValue(e.GitBranch),
				"id":         types.StringValue(e.ID),
				"sensitive":  types.BoolValue(e.Type == "sensitive"),
				"comment":    types.StringValue(e.Comment),
			},
		))
	}

	return ProjectEnvironmentVariables{
		TeamID:    toTeamID(plan.TeamID.ValueString()),
		ProjectID: plan.ProjectID,
		Variables: types.SetValueMust(envVariableElemType, env),
	}, nil
}

// Create will create a new project environment variable for a Vercel project.
// This is called automatically by the provider when a new resource should be created.
func (r *projectEnvironmentVariablesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectEnvironmentVariables
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.GetProject(ctx, plan.ProjectID.ValueString(), plan.TeamID.ValueString())
	if client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Error creating project environment variables",
			"Could not find project, please make sure both the project_id and team_id match the project and team you wish to deploy to.",
		)
		return
	}

	request, err := plan.toCreateEnvironmentVariablesRequest(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project environment variables",
			"Could not create project environment variables request, unexpected error: "+err.Error(),
		)
		return
	}
	created, err := r.client.CreateEnvironmentVariables(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project environment variables",
			"Could not create project environment variables, unexpected error: "+err.Error(),
		)
	}

	result, err := convertResponseToProjectEnvironmentVariables(ctx, created, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project environment variables",
			"Could not read environment variables, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "created project environment variables", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
		"variables":  created,
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read an environment variable of a Vercel project by requesting it from the Vercel API, and will update terraform
// with this information.
func (r *projectEnvironmentVariablesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectEnvironmentVariables
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	existing, err := state.environment(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project environment variables",
			"Could not read environment variables, unexpected error: "+err.Error(),
		)
		return
	}
	existingIDs := map[string]struct{}{}
	for _, e := range existing {
		if e.ID.ValueString() == "" {
			existingIDs[e.ID.ValueString()] = struct{}{}
		}
	}
	if len(existingIDs) == 0 {
		// no existing environment variables, nothing to do
		return
	}

	envs, err := r.client.ListEnvironmentVariables(ctx, state.TeamID.ValueString(), state.ProjectID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project environment variables",
			"Could not read environment variables, unexpected error: "+err.Error(),
		)
		return
	}

	var toUse []client.EnvironmentVariable
	for _, e := range envs {
		if _, ok := existingIDs[e.ID]; ok {
			toUse = append(toUse, e)
		}
	}

	result, err := convertResponseToProjectEnvironmentVariables(ctx, toUse, state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project environment variables",
			"Could not read environment variables, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "read project environment variables", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the project environment variable of a Vercel project state.
func (r *projectEnvironmentVariablesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProjectEnvironmentVariables
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ProjectEnvironmentVariables
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stateEnvs, err := state.environment(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project environment variables",
			"Could not read environment variables, unexpected error: "+err.Error(),
		)
		return
	}
	planEnvs, err := plan.environment(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project environment variables",
			"Could not read environment variables, unexpected error: "+err.Error(),
		)
		return
	}
	plannedIDs := map[string]struct{}{}
	for _, e := range planEnvs {
		if e.ID.ValueString() == "" {
			plannedIDs[e.ID.ValueString()] = struct{}{}
		}
	}

	var toRemove []EnvironmentItem
	for _, e := range stateEnvs {
		if _, ok := plannedIDs[e.ID.ValueString()]; !ok {
			toRemove = append(toRemove, e)
		}
	}

	for _, v := range toRemove {
		err := r.client.DeleteEnvironmentVariable(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString(), v.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating Project Environment Variables",
				fmt.Sprintf(
					"Could not remove environment variable %s (%s), unexpected error: %s",
					v.Key.ValueString(),
					v.ID.ValueString(),
					err,
				),
			)
			return
		}
		tflog.Info(ctx, "deleted environment variable", map[string]interface{}{
			"team_id":        plan.TeamID.ValueString(),
			"project_id":     plan.ProjectID.ValueString(),
			"environment_id": v.ID.ValueString(),
		})
	}

	request, err := plan.toCreateEnvironmentVariablesRequest(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project environment variables",
			"Could not create project environment variables request, unexpected error: "+err.Error(),
		)
		return
	}
	response, err := r.client.CreateEnvironmentVariables(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project environment variables",
			"Could not update project environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	result, err := convertResponseToProjectEnvironmentVariables(ctx, response, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project environment variables",
			"Could not read environment variables, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "updated project environment variables", map[string]interface{}{
		"team_id":    result.TeamID.ValueString(),
		"project_id": result.ProjectID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes a Vercel project environment variable.
func (r *projectEnvironmentVariablesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectEnvironmentVariables
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	envs, err := state.environment(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing project environment variables",
			"Could not read environment variables, unexpected error: "+err.Error(),
		)
		return
	}
	for _, v := range envs {
		err := r.client.DeleteEnvironmentVariable(ctx, state.ProjectID.ValueString(), state.TeamID.ValueString(), v.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating Project Environment Variables",
				fmt.Sprintf(
					"Could not remove environment variable %s (%s), unexpected error: %s",
					v.Key.ValueString(),
					v.ID.ValueString(),
					err,
				),
			)
			return
		}
		tflog.Info(ctx, "deleted environment variable", map[string]interface{}{
			"team_id":        state.TeamID.ValueString(),
			"project_id":     state.ProjectID.ValueString(),
			"environment_id": v.ID.ValueString(),
		})
	}
}
