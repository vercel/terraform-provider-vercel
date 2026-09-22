package vercel

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var (
	_ resource.Resource                   = &projectDeploymentCheckResource{}
	_ resource.ResourceWithConfigure      = &projectDeploymentCheckResource{}
	_ resource.ResourceWithImportState    = &projectDeploymentCheckResource{}
	_ resource.ResourceWithModifyPlan     = &projectDeploymentCheckResource{}
	_ resource.ResourceWithValidateConfig = &projectDeploymentCheckResource{}
)

func newProjectDeploymentCheckResource() resource.Resource {
	return &projectDeploymentCheckResource{}
}

type projectDeploymentCheckResource struct {
	client *client.Client
}

func (r *projectDeploymentCheckResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_deployment_check"
}

func (r *projectDeploymentCheckResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	configuredClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}
	r.client = configuredClient
}

func (r *projectDeploymentCheckResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Creates and manages a Deployment Check for a Vercel project.",
		Attributes: map[string]schema.Attribute{
			"id":               schema.StringAttribute{Computed: true, MarkdownDescription: "The ID of the Deployment Check.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()}},
			"project_id":       schema.StringAttribute{Required: true, MarkdownDescription: "The ID or name of the Vercel project.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"team_id":          schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "The ID of the Vercel team.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()}},
			"name":             schema.StringAttribute{Required: true, MarkdownDescription: "The human-readable name of the Deployment Check.", Validators: []validator.String{stringvalidator.LengthAtLeast(1), validateStringIsTrimmed()}},
			"requires":         schema.StringAttribute{Required: true, MarkdownDescription: "The deployment stage required before the check runs: `build-ready`, `deployment-url`, or `none`. Changing this to `none` replaces the check because the API does not support that update.", Validators: []validator.String{stringvalidator.OneOf("build-ready", "deployment-url", "none")}},
			"is_rerequestable": schema.BoolAttribute{Optional: true, Computed: true, MarkdownDescription: "Whether users can rerun the check. Defaults to false; must be false for a `git-provider` source.", PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseNonNullStateForUnknown()}},
			"blocks":           schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "The deployment stage blocked by the check. New checks currently support `deployment-alias` and `none`.", Validators: []validator.String{stringvalidator.OneOf("deployment-alias", "none")}, PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()}},
			"targets":          schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType, MarkdownDescription: "Deployment environment slugs to which the check applies, such as `production`, `preview`, or a custom environment slug. Use `[\"all\"]` for every environment; `all` cannot be combined with other targets. The API defaults to `[\"production\"]`.", Validators: []validator.Set{setvalidator.SizeAtLeast(1), setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1), validateStringIsTrimmed())}, PlanModifiers: []planmodifier.Set{setplanmodifier.UseNonNullStateForUnknown()}},
			"timeout":          schema.Int64Attribute{Optional: true, Computed: true, MarkdownDescription: "The timeout value supplied to check runners by the Checks API. When omitted, the API determines the value.", Validators: []validator.Int64{int64validator.AtLeast(1)}, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseNonNullStateForUnknown()}},
			"source": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The system that supplies the Deployment Check. Required when creating a check; changing a configured source field replaces the check. Omit it when managing an imported check whose source is not writable through the API.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseNonNullStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"kind":                 schema.StringAttribute{Required: true, MarkdownDescription: "The source kind. New checks support `git-provider`, `integration`, and `webhook`; `vercel` is response-only.", Validators: []validator.String{stringvalidator.OneOf("git-provider", "integration", "webhook", "vercel")}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
					"external_check_name":  schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "The external check name. Required for a `git-provider` source.", Validators: []validator.String{stringvalidator.LengthAtLeast(1), validateStringIsTrimmed()}, PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown(), stringplanmodifier.RequiresReplaceIfConfigured()}},
					"provider":             schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "The Git provider. New git-provider checks currently support `github`; imported checks may report `gitlab` or `bitbucket`.", Validators: []validator.String{stringvalidator.OneOf("github", "gitlab", "bitbucket")}, PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown(), stringplanmodifier.RequiresReplaceIfConfigured()}},
					"webhook_id":           schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "The webhook ID for a `webhook` source.", Validators: []validator.String{stringvalidator.LengthAtLeast(1), validateStringIsTrimmed()}, PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown(), stringplanmodifier.RequiresReplaceIfConfigured()}},
					"external_resource_id": schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "An optional external resource ID for an `integration` source. Creating integration checks requires an integration token; the API derives integration IDs from that token.", Validators: []validator.String{stringvalidator.LengthAtLeast(1), validateStringIsTrimmed()}, PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown(), stringplanmodifier.RequiresReplaceIfConfigured()}},
				},
			},
		},
	}
}

type ProjectDeploymentCheck struct {
	ID              types.String `tfsdk:"id"`
	ProjectID       types.String `tfsdk:"project_id"`
	TeamID          types.String `tfsdk:"team_id"`
	Name            types.String `tfsdk:"name"`
	Requires        types.String `tfsdk:"requires"`
	IsRerequestable types.Bool   `tfsdk:"is_rerequestable"`
	Blocks          types.String `tfsdk:"blocks"`
	Targets         types.Set    `tfsdk:"targets"`
	Timeout         types.Int64  `tfsdk:"timeout"`
	Source          types.Object `tfsdk:"source"`
}

type ProjectDeploymentCheckSource struct {
	Kind               types.String `tfsdk:"kind"`
	ExternalCheckName  types.String `tfsdk:"external_check_name"`
	Provider           types.String `tfsdk:"provider"`
	WebhookID          types.String `tfsdk:"webhook_id"`
	ExternalResourceID types.String `tfsdk:"external_resource_id"`
}

var projectDeploymentCheckSourceAttrTypes = map[string]attr.Type{
	"kind": types.StringType, "external_check_name": types.StringType, "provider": types.StringType,
	"webhook_id": types.StringType, "external_resource_id": types.StringType,
}

func (r *projectDeploymentCheckResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config ProjectDeploymentCheck
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if !config.Targets.IsNull() && !config.Targets.IsUnknown() {
		allKnown := true
		for _, target := range config.Targets.Elements() {
			allKnown = allKnown && !target.IsUnknown()
		}
		for _, target := range config.Targets.Elements() {
			if allKnown && target.Equal(types.StringValue("all")) && len(config.Targets.Elements()) > 1 {
				resp.Diagnostics.AddAttributeError(path.Root("targets"), "Invalid deployment targets", "The all target cannot be combined with other environment slugs.")
			}
		}
	}
	if resp.Diagnostics.HasError() || config.Source.IsNull() || config.Source.IsUnknown() {
		return
	}
	var source ProjectDeploymentCheckSource
	resp.Diagnostics.Append(config.Source.As(ctx, &source, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() || source.Kind.IsUnknown() {
		return
	}
	if source.Kind.ValueString() == "git-provider" {
		if config.IsRerequestable.ValueBool() {
			resp.Diagnostics.AddAttributeError(path.Root("is_rerequestable"), "Unsupported rerun setting", "Git provider Deployment Checks cannot be rerequestable. Set is_rerequestable to false or omit it.")
		}
		if source.ExternalCheckName.IsNull() {
			resp.Diagnostics.AddError("Missing source.external_check_name", "source.external_check_name is required when source.kind is git-provider.")
		}
		if source.Provider.IsNull() {
			resp.Diagnostics.AddError("Missing source.provider", "source.provider is required when source.kind is git-provider.")
		} else if !source.Provider.IsUnknown() && source.Provider.ValueString() != "github" {
			resp.Diagnostics.AddError("Unsupported source.provider", "Only github can be configured for a new git-provider Deployment Check.")
		}
	}
	for _, field := range []struct {
		name  string
		kind  string
		value types.String
	}{
		{"external_check_name", "git-provider", source.ExternalCheckName},
		{"provider", "git-provider", source.Provider},
		{"webhook_id", "webhook", source.WebhookID},
		{"external_resource_id", "integration", source.ExternalResourceID},
	} {
		if !field.value.IsNull() && source.Kind.ValueString() != field.kind {
			resp.Diagnostics.AddAttributeError(path.Root("source").AtName(field.name), "Invalid source attribute", fmt.Sprintf("source.%s can only be configured for a %s source.", field.name, field.kind))
		}
	}
	if source.Kind.ValueString() == "vercel" {
		resp.Diagnostics.AddError("Unsupported source.kind", "Vercel-managed Deployment Checks can be imported, but cannot be created by this resource.")
	}
}

func (r *projectDeploymentCheckResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var config ProjectDeploymentCheck
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if req.State.Raw.IsNull() && config.Source.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("source"), "Missing Deployment Check source", "A source is required when creating a project Deployment Check.")
	}
	if !req.State.Raw.IsNull() {
		var state, plan ProjectDeploymentCheck
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
		// PATCH cannot set requires to none, although POST supports it.
		if plan.Requires.ValueString() == "none" && !plan.Requires.Equal(state.Requires) {
			resp.RequiresReplace = append(resp.RequiresReplace, path.Root("requires"))
			if config.Source.IsNull() {
				resp.Diagnostics.AddAttributeError(path.Root("source"), "Missing replacement source", "Changing requires to none replaces the check. Configure a writable source to create the replacement.")
			}
		}
	}
}

func projectDeploymentCheckSourceToClient(ctx context.Context, value types.Object) (client.ProjectDeploymentCheckSource, diag.Diagnostics) {
	var source ProjectDeploymentCheckSource
	diags := value.As(ctx, &source, basetypes.ObjectAsOptions{})
	return client.ProjectDeploymentCheckSource{
		Kind: source.Kind.ValueString(), ExternalCheckName: source.ExternalCheckName.ValueString(), Provider: source.Provider.ValueString(),
		WebhookID: source.WebhookID.ValueString(), ExternalResourceID: source.ExternalResourceID.ValueString(),
	}, diags
}

func projectDeploymentCheckSourceFromClient(ctx context.Context, source client.ProjectDeploymentCheckSource) (types.Object, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, projectDeploymentCheckSourceAttrTypes, ProjectDeploymentCheckSource{
		Kind: types.StringValue(source.Kind), ExternalCheckName: optionalStringValue(source.ExternalCheckName), Provider: optionalStringValue(source.Provider),
		WebhookID: optionalStringValue(source.WebhookID), ExternalResourceID: optionalStringValue(source.ExternalResourceID),
	})
}

func optionalStringValue(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func projectDeploymentCheckFromClient(ctx context.Context, check client.ProjectDeploymentCheck, projectID, teamID types.String) (ProjectDeploymentCheck, diag.Diagnostics) {
	source, diags := projectDeploymentCheckSourceFromClient(ctx, check.Source)
	targets, targetDiags := types.SetValueFrom(ctx, types.StringType, check.Targets)
	diags.Append(targetDiags...)
	return ProjectDeploymentCheck{
		ID: types.StringValue(check.ID), ProjectID: projectID, TeamID: teamID,
		Name: types.StringValue(check.Name), Requires: types.StringValue(check.Requires), IsRerequestable: types.BoolValue(check.IsRerequestable),
		Blocks: types.StringValue(check.Blocks), Targets: targets, Timeout: types.Int64Value(check.Timeout),
		Source: source,
	}, diags
}

func (model ProjectDeploymentCheck) createRequest(ctx context.Context) (client.CreateProjectDeploymentCheckRequest, diag.Diagnostics) {
	request := client.CreateProjectDeploymentCheckRequest{
		ProjectID: model.ProjectID.ValueString(), TeamID: model.TeamID.ValueString(), Name: model.Name.ValueString(),
		Requires: model.Requires.ValueString(),
	}
	var diags diag.Diagnostics
	if !model.Source.IsNull() && !model.Source.IsUnknown() {
		source, sourceDiags := projectDeploymentCheckSourceToClient(ctx, model.Source)
		diags.Append(sourceDiags...)
		request.Source = &source
	}
	if !model.IsRerequestable.IsNull() && !model.IsRerequestable.IsUnknown() {
		value := model.IsRerequestable.ValueBool()
		request.IsRerequestable = &value
	}
	if !model.Blocks.IsNull() && !model.Blocks.IsUnknown() {
		value := model.Blocks.ValueString()
		request.Blocks = &value
	}
	if !model.Targets.IsNull() && !model.Targets.IsUnknown() {
		var targets []string
		diags.Append(model.Targets.ElementsAs(ctx, &targets, false)...)
		request.Targets = &targets
	}
	if !model.Timeout.IsNull() && !model.Timeout.IsUnknown() {
		value := model.Timeout.ValueInt64()
		request.Timeout = &value
	}
	return request, diags
}

func (plan ProjectDeploymentCheck) updateRequest(ctx context.Context, state ProjectDeploymentCheck) (client.UpdateProjectDeploymentCheckRequest, diag.Diagnostics) {
	request := client.UpdateProjectDeploymentCheckRequest{ProjectID: plan.ProjectID.ValueString(), TeamID: plan.TeamID.ValueString(), ID: state.ID.ValueString()}
	var diags diag.Diagnostics
	if !plan.Name.Equal(state.Name) {
		value := plan.Name.ValueString()
		request.Name = &value
	}
	if !plan.Requires.Equal(state.Requires) {
		value := plan.Requires.ValueString()
		request.Requires = &value
	}
	if !plan.IsRerequestable.Equal(state.IsRerequestable) {
		value := plan.IsRerequestable.ValueBool()
		request.IsRerequestable = &value
	}
	if !plan.Blocks.Equal(state.Blocks) {
		value := plan.Blocks.ValueString()
		request.Blocks = &value
	}
	if !plan.Timeout.Equal(state.Timeout) {
		value := plan.Timeout.ValueInt64()
		request.Timeout = &value
	}
	if !plan.Targets.Equal(state.Targets) {
		var targets []string
		diags.Append(plan.Targets.ElementsAs(ctx, &targets, false)...)
		request.Targets = &targets
	}
	return request, diags
}

func (r *projectDeploymentCheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectDeploymentCheck
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	request, diags := plan.createRequest(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	check, err := r.client.CreateProjectDeploymentCheck(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Error creating project deployment check", err.Error())
		return
	}
	result, diags := projectDeploymentCheckFromClient(ctx, check, plan.ProjectID, toTeamID(r.client.TeamID(plan.TeamID.ValueString())))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "created project deployment check", map[string]any{"id": check.ID, "project_id": check.ProjectID})
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *projectDeploymentCheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectDeploymentCheck
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	check, err := r.client.GetProjectDeploymentCheck(ctx, state.ProjectID.ValueString(), state.ID.ValueString(), state.TeamID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading project deployment check", err.Error())
		return
	}
	result, diags := projectDeploymentCheckFromClient(ctx, check, state.ProjectID, toTeamID(r.client.TeamID(state.TeamID.ValueString())))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *projectDeploymentCheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ProjectDeploymentCheck
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	request, diags := plan.updateRequest(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	check, err := r.client.UpdateProjectDeploymentCheck(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Error updating project deployment check", err.Error())
		return
	}
	result, diags := projectDeploymentCheckFromClient(ctx, check, plan.ProjectID, toTeamID(r.client.TeamID(plan.TeamID.ValueString())))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *projectDeploymentCheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectDeploymentCheck
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.DeleteProjectDeploymentCheck(ctx, state.ProjectID.ValueString(), state.ID.ValueString(), state.TeamID.ValueString())
	if err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Error deleting project deployment check", err.Error())
		return
	}
}

func (r *projectDeploymentCheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, checkID, ok := splitInto2Or3(req.ID)
	if !ok || projectID == "" || checkID == "" || strings.HasPrefix(req.ID, "/") || strings.Contains(req.ID, "//") {
		resp.Diagnostics.AddError("Error importing project deployment check", fmt.Sprintf("Invalid id %q. Expected team_id/project_id/check_id or project_id/check_id.", req.ID))
		return
	}
	check, err := r.client.GetProjectDeploymentCheck(ctx, projectID, checkID, teamID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing project deployment check", err.Error())
		return
	}
	resolvedTeamID := r.client.TeamID(teamID)
	if resolvedTeamID == "" && strings.HasPrefix(check.OwnerID, "team_") {
		resolvedTeamID = check.OwnerID
	}
	result, diags := projectDeploymentCheckFromClient(ctx, check, types.StringValue(projectID), toTeamID(resolvedTeamID))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}
