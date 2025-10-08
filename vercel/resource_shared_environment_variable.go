package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

var (
	_ resource.Resource                     = &sharedEnvironmentVariableResource{}
	_ resource.ResourceWithConfigure        = &sharedEnvironmentVariableResource{}
	_ resource.ResourceWithImportState      = &sharedEnvironmentVariableResource{}
	_ resource.ResourceWithModifyPlan       = &sharedEnvironmentVariableResource{}
	_ resource.ResourceWithConfigValidators = &sharedEnvironmentVariableResource{}
)

func newSharedEnvironmentVariableResource() resource.Resource {
	return &sharedEnvironmentVariableResource{}
}

type sharedEnvironmentVariableResource struct {
	client *client.Client
}

func (r *sharedEnvironmentVariableResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var config SharedEnvironmentVariable
	diags := req.Plan.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.ID.ValueString() != "" {
		// The resource already exists, so this is okay.
		return
	}
	if config.Sensitive.IsUnknown() || config.Sensitive.IsNull() || config.Sensitive.ValueBool() {
		// Sensitive is either true, or computed, which is fine.
		return
	}

	// if sensitive is explicitly set to `false`, then validate that an env var can be created with the given
	// team sensitive environment variable policy.
	team, err := r.client.Team(ctx, config.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error validating shared environment variable",
			"Could not validate shared environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	if team.SensitiveEnvironmentVariablePolicy == nil || *team.SensitiveEnvironmentVariablePolicy != "on" {
		// the policy isn't enabled
		return
	}

	resp.Diagnostics.AddAttributeError(
		path.Root("sensitive"),
		"Shared Environment Variable Invalid",
		"This team has a policy that forces all environment variables to be sensitive. Please remove the `sensitive` field or set the `sensitive` field to `true` in your configuration.",
	)
}

func (r *sharedEnvironmentVariableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shared_environment_variable"
}

func (r *sharedEnvironmentVariableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Schema returns the schema information for a shared environment variable resource.
func (r *sharedEnvironmentVariableResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Shared Environment Variable resource.

A Shared Environment Variable resource defines an Environment Variable that can be shared between multiple Vercel Projects.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/concepts/projects/environment-variables/shared-environment-variables).
`,
		Attributes: map[string]schema.Attribute{
			"target": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The environments that the Environment Variable should be present on. Valid targets are either `production`, `preview`, or `development`.",
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.OneOf("production", "preview", "development")),
				},
			},
			"key": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "The name of the Environment Variable.",
			},
			"value": schema.StringAttribute{
				Required:    true,
				Description: "The value of the Environment Variable.",
				Sensitive:   true,
			},
			"project_ids": schema.SetAttribute{
				Required:    true,
				Description: "The ID of the Vercel project.",
				ElementType: types.StringType,
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the Vercel team. Shared environment variables require a team.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"id": schema.StringAttribute{
				Description:   "The ID of the Environment Variable.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
				Computed:      true,
			},
			"sensitive": schema.BoolAttribute{
				Description:   "Whether the Environment Variable is sensitive or not. (May be affected by a [team-wide environment variable policy](https://vercel.com/docs/projects/environment-variables/sensitive-environment-variables#environment-variables-policy))",
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
			"apply_to_all_custom_environments": schema.BoolAttribute{
				Description: "Whether the shared environment variable should be applied to all custom environments in the linked projects.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *sharedEnvironmentVariableResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		&sharedEnvTargetValidator{},
	}
}

type sharedEnvTargetValidator struct{}

func (v *sharedEnvTargetValidator) Description(ctx context.Context) string {
	return "When `apply_to_all_custom_environments` is `false` or not set, you must specify `target` with at least one value."
}

func (v *sharedEnvTargetValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *sharedEnvTargetValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	// Read only the required attributes to avoid decoding Unknowns.
	var applyAll types.Bool
	diags := req.Config.GetAttribute(ctx, path.Root("apply_to_all_custom_environments"), &applyAll)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If apply_to_all_custom_environments is unknown (computed), skip validation
	// since we can't determine the configuration's validity during planning.
	if applyAll.IsUnknown() {
		return
	}

	// If apply_to_all_custom_environments is explicitly true, allow target to be omitted or empty.
	if !applyAll.IsNull() && applyAll.ValueBool() {
		return
	}

	// Read target attribute without decoding the entire resource.
	var target types.Set
	diags = req.Config.GetAttribute(ctx, path.Root("target"), &target)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If target is unknown, skip validation at this stage.
	if target.IsUnknown() {
		return
	}

	// Otherwise, target must be provided with at least one element.
	if target.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("target"),
			"Missing required attribute",
			v.Description(ctx),
		)
		return
	}

	var targets []string
	diags = target.ElementsAs(ctx, &targets, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(targets) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("target"),
			"Invalid attribute value",
			v.Description(ctx),
		)
	}
}

// SharedEnvironmentVariable reflects the state terraform stores internally for a project environment variable.
type SharedEnvironmentVariable struct {
	Target                       types.Set    `tfsdk:"target"`
	Key                          types.String `tfsdk:"key"`
	Value                        types.String `tfsdk:"value"`
	TeamID                       types.String `tfsdk:"team_id"`
	ProjectIDs                   types.Set    `tfsdk:"project_ids"`
	ID                           types.String `tfsdk:"id"`
	Sensitive                    types.Bool   `tfsdk:"sensitive"`
	Comment                      types.String `tfsdk:"comment"`
	ApplyToAllCustomEnvironments types.Bool   `tfsdk:"apply_to_all_custom_environments"`
}

func (e *SharedEnvironmentVariable) toCreateSharedEnvironmentVariableRequest(ctx context.Context, diags diag.Diagnostics) (req client.CreateSharedEnvironmentVariableRequest, ok bool) {
	var target []string
	if e.Target.IsNull() || e.Target.IsUnknown() {
		target = []string{}
	} else {
		ds := e.Target.ElementsAs(ctx, &target, false)
		diags = append(diags, ds...)
		if diags.HasError() {
			return req, false
		}
	}

	var projectIDs []string
	ds := e.ProjectIDs.ElementsAs(ctx, &projectIDs, false)
	diags = append(diags, ds...)
	if diags.HasError() {
		return req, false
	}

	var envVariableType string

	if e.Sensitive.ValueBool() {
		envVariableType = "sensitive"
	} else {
		envVariableType = "encrypted"
	}

	return client.CreateSharedEnvironmentVariableRequest{
		EnvironmentVariable: client.SharedEnvironmentVariableRequest{
			ApplyToAllCustomEnvironments: e.ApplyToAllCustomEnvironments.ValueBool(),
			Target:                       target,
			Type:                         envVariableType,
			ProjectIDs:                   projectIDs,
			EnvironmentVariables: []client.SharedEnvVarRequest{
				{
					Key:     e.Key.ValueString(),
					Value:   e.Value.ValueString(),
					Comment: e.Comment.ValueString(),
				},
			},
		},
		TeamID: e.TeamID.ValueString(),
	}, true
}

func (e *SharedEnvironmentVariable) toUpdateSharedEnvironmentVariableRequest(ctx context.Context, diags diag.Diagnostics) (req client.UpdateSharedEnvironmentVariableRequest, ok bool) {
	var target []string
	if e.Target.IsNull() || e.Target.IsUnknown() {
		target = []string{}
	} else {
		ds := e.Target.ElementsAs(ctx, &target, false)
		diags = append(diags, ds...)
		if diags.HasError() {
			return req, false
		}
	}

	var projectIDs []string
	ds := e.ProjectIDs.ElementsAs(ctx, &projectIDs, false)
	diags = append(diags, ds...)
	if diags.HasError() {
		return req, false
	}
	var envVariableType string

	if e.Sensitive.ValueBool() {
		envVariableType = "sensitive"
	} else {
		envVariableType = "encrypted"
	}
	return client.UpdateSharedEnvironmentVariableRequest{
		ApplyToAllCustomEnvironments: e.ApplyToAllCustomEnvironments.ValueBool(),
		Value:                        e.Value.ValueString(),
		Target:                       target,
		Type:                         envVariableType,
		TeamID:                       e.TeamID.ValueString(),
		EnvID:                        e.ID.ValueString(),
		ProjectIDs:                   projectIDs,
		Comment:                      e.Comment.ValueString(),
	}, true
}

// convertResponseToSharedEnvironmentVariable is used to populate terraform state based on an API response.
// Where possible, values from the API response are used to populate state. If not possible,
// values from plan are used.
func convertResponseToSharedEnvironmentVariable(response client.SharedEnvironmentVariableResponse, v types.String) SharedEnvironmentVariable {
	target := []attr.Value{}
	for _, t := range response.Target {
		target = append(target, types.StringValue(t))
	}

	projectIDs := []attr.Value{}
	for _, t := range response.ProjectIDs {
		projectIDs = append(projectIDs, types.StringValue(t))
	}

	value := types.StringValue(response.Value)
	if response.Type == "sensitive" {
		value = v
	}

	return SharedEnvironmentVariable{
		ApplyToAllCustomEnvironments: types.BoolValue(response.ApplyToAllCustomEnvironments),
		Target:                       types.SetValueMust(types.StringType, target),
		Key:                          types.StringValue(response.Key),
		Value:                        value,
		ProjectIDs:                   types.SetValueMust(types.StringType, projectIDs),
		TeamID:                       toTeamID(response.TeamID),
		ID:                           types.StringValue(response.ID),
		Sensitive:                    types.BoolValue(response.Type == "sensitive"),
		Comment:                      types.StringValue(response.Comment),
	}
}

// Create will create a new shared environment variable.
// This is called automatically by the provider when a new resource should be created.
func (r *sharedEnvironmentVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SharedEnvironmentVariable
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request, ok := plan.toCreateSharedEnvironmentVariableRequest(ctx, resp.Diagnostics)
	if !ok {
		return
	}
	response, err := r.client.CreateSharedEnvironmentVariable(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating shared environment variable",
			"Could not create shared environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToSharedEnvironmentVariable(response, plan.Value)

	tflog.Info(ctx, "created shared environment variable", map[string]any{
		"id":      result.ID.ValueString(),
		"team_id": result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read an shared environment variable by requesting it from the Vercel API, and will update terraform
// with this information.
func (r *sharedEnvironmentVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SharedEnvironmentVariable
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetSharedEnvironmentVariable(ctx, state.TeamID.ValueString(), state.ID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading shared environment variable",
			fmt.Sprintf("Could not get shared environment variable %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToSharedEnvironmentVariable(out, state.Value)
	tflog.Info(ctx, "read shared environment variable", map[string]any{
		"id":      result.ID.ValueString(),
		"team_id": result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the shared environment variable of a Vercel project state.
func (r *sharedEnvironmentVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SharedEnvironmentVariable
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	request, ok := plan.toUpdateSharedEnvironmentVariableRequest(ctx, resp.Diagnostics)
	if !ok {
		return
	}
	response, err := r.client.UpdateSharedEnvironmentVariable(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating shared environment variable",
			"Could not update shared environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToSharedEnvironmentVariable(response, plan.Value)

	tflog.Info(ctx, "updated shared environment variable", map[string]any{
		"id":      result.ID.ValueString(),
		"team_id": result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes a Vercel shared environment variable.
func (r *sharedEnvironmentVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SharedEnvironmentVariable
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSharedEnvironmentVariable(ctx, state.TeamID.ValueString(), state.ID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting shared environment variable",
			fmt.Sprintf(
				"Could not delete shared environment variable %s, unexpected error: %s",
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted shared environment variable", map[string]any{
		"id":      state.ID.ValueString(),
		"team_id": state.TeamID.ValueString(),
	})
}

// ImportState takes an identifier and reads all the shared environment variable information from the Vercel API.
// The results are then stored in terraform state.
func (r *sharedEnvironmentVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, envID, ok := splitInto1Or2(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing shared environment variable",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/env_id\"", req.ID),
		)
	}

	out, err := r.client.GetSharedEnvironmentVariable(ctx, teamID, envID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading shared environment variable",
			fmt.Sprintf("Could not get shared environment variable %s %s, unexpected error: %s",
				teamID,
				envID,
				err,
			),
		)
		return
	}

	result := convertResponseToSharedEnvironmentVariable(out, types.StringNull())
	tflog.Info(ctx, "imported shared environment variable", map[string]any{
		"team_id": result.TeamID.ValueString(),
		"env_id":  result.ID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
