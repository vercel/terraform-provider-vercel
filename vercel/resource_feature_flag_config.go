package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var (
	_ resource.Resource                = &featureFlagConfigResource{}
	_ resource.ResourceWithConfigure   = &featureFlagConfigResource{}
	_ resource.ResourceWithImportState = &featureFlagConfigResource{}
)

func newFeatureFlagConfigResource() resource.Resource {
	return &featureFlagConfigResource{}
}

type featureFlagConfigResource struct {
	client *client.Client
}

type featureFlagConfigModel struct {
	ID          types.String                `tfsdk:"id"`
	ProjectID   types.String                `tfsdk:"project_id"`
	TeamID      types.String                `tfsdk:"team_id"`
	FlagID      types.String                `tfsdk:"flag_id"`
	Production  featureFlagEnvironmentModel `tfsdk:"production"`
	Preview     featureFlagEnvironmentModel `tfsdk:"preview"`
	Development featureFlagEnvironmentModel `tfsdk:"development"`
}

func (r *featureFlagConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature_flag_config"
}

func (r *featureFlagConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func featureFlagConfigEnvironmentSchema(description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: description,
		Required:    true,
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the flag should actively evaluate in this environment.",
			},
			"default_variant_id": schema.StringAttribute{
				Required:    true,
				Description: "The variant to serve when this environment is enabled and no rules match.",
			},
			"disabled_variant_id": schema.StringAttribute{
				Required:    true,
				Description: "The variant to serve while this environment is disabled or paused.",
			},
		},
	}
}

func (r *featureFlagConfigResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Feature Flag Config resource.

This resource manages the simplified static rollout shape for a flag across ` + "`production`" + `, ` + "`preview`" + `, and ` + "`development`" + `.

Use this resource together with ` + "`vercel_feature_flag_definition`" + ` when Terraform should own the rollout. If you omit this resource, the flag definition can still exist while rollout is managed through the Vercel dashboard.

It is intentionally strict: linked environments, rules, and target overrides must not already be configured on the flag when Terraform manages this resource.

Deleting this resource only removes it from Terraform state. The flag and its current configuration stay in Vercel.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the feature flag whose config is managed.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the Vercel project that owns the flag.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the Vercel team. Required when configuring a team resource if a default team has not been set in the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseNonNullStateForUnknown()},
			},
			"flag_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the feature flag whose config Terraform should manage.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"production":  featureFlagConfigEnvironmentSchema("The production environment behavior for this flag."),
			"preview":     featureFlagConfigEnvironmentSchema("The preview environment behavior for this flag."),
			"development": featureFlagConfigEnvironmentSchema("The development environment behavior for this flag."),
		},
	}
}

func (r *featureFlagConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan featureFlagConfigModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.TeamID = types.StringValue(r.client.TeamID(plan.TeamID.ValueString()))

	result, diags := r.applyFeatureFlagConfig(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "created feature flag config", map[string]any{
		"project_id": result.ProjectID.ValueString(),
		"team_id":    result.TeamID.ValueString(),
		"flag_id":    result.FlagID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *featureFlagConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state featureFlagConfigModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.TeamID = types.StringValue(r.client.TeamID(state.TeamID.ValueString()))

	out, err := r.client.GetFeatureFlag(ctx, client.GetFeatureFlagRequest{
		ProjectID: state.ProjectID.ValueString(),
		TeamID:    state.TeamID.ValueString(),
		FlagID:    state.FlagID.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Feature Flag Config",
			fmt.Sprintf(
				"Could not get Feature Flag Config %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ProjectID.ValueString(),
				state.FlagID.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := featureFlagConfigFromClient(out, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *featureFlagConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan featureFlagConfigModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.TeamID = types.StringValue(r.client.TeamID(plan.TeamID.ValueString()))

	result, diags := r.applyFeatureFlagConfig(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *featureFlagConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state featureFlagConfigModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "removed feature flag config from terraform state", map[string]any{
		"project_id": state.ProjectID.ValueString(),
		"team_id":    state.TeamID.ValueString(),
		"flag_id":    state.FlagID.ValueString(),
	})
}

func (r *featureFlagConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, flagID, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Feature Flag Config",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/project_id/flag_id\" or \"project_id/flag_id\"", req.ID),
		)
		return
	}

	out, err := r.client.GetFeatureFlag(ctx, client.GetFeatureFlagRequest{
		ProjectID: projectID,
		TeamID:    teamID,
		FlagID:    flagID,
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing Feature Flag Config",
			fmt.Sprintf("Could not get Feature Flag Config %s %s %s, unexpected error: %s", teamID, projectID, flagID, err),
		)
		return
	}

	result, diags := featureFlagConfigFromClient(out, featureFlagConfigModel{
		ProjectID: types.StringValue(projectID),
		TeamID:    types.StringValue(r.client.TeamID(teamID)),
		FlagID:    types.StringValue(flagID),
		ID:        types.StringValue(flagID),
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *featureFlagConfigResource) applyFeatureFlagConfig(ctx context.Context, plan featureFlagConfigModel) (featureFlagConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	out, err := r.client.GetFeatureFlag(ctx, client.GetFeatureFlagRequest{
		ProjectID: plan.ProjectID.ValueString(),
		TeamID:    plan.TeamID.ValueString(),
		FlagID:    plan.FlagID.ValueString(),
	})
	if client.NotFound(err) {
		diags.AddError(
			"Feature Flag not found",
			fmt.Sprintf("Could not find Feature Flag %s %s %s.", plan.TeamID.ValueString(), plan.ProjectID.ValueString(), plan.FlagID.ValueString()),
		)
		return featureFlagConfigModel{}, diags
	}
	if err != nil {
		diags.AddError(
			"Error reading Feature Flag before applying config",
			fmt.Sprintf(
				"Could not get Feature Flag %s %s %s before applying config, unexpected error: %s",
				plan.TeamID.ValueString(),
				plan.ProjectID.ValueString(),
				plan.FlagID.ValueString(),
				err,
			),
		)
		return featureFlagConfigModel{}, diags
	}

	diags.Append(featureFlagConfigValidateExistingEnvironments(out)...)
	if diags.HasError() {
		return featureFlagConfigModel{}, diags
	}

	environments, d := featureFlagEnvironmentsFromModel(map[string]featureFlagEnvironmentModel{
		"production":  plan.Production,
		"preview":     plan.Preview,
		"development": plan.Development,
	}, featureFlagVariantIDsFromClient(out.Variants))
	diags.Append(d...)
	if diags.HasError() {
		return featureFlagConfigModel{}, diags
	}

	updated, err := r.client.UpdateFeatureFlag(ctx, client.UpdateFeatureFlagRequest{
		ProjectID:    plan.ProjectID.ValueString(),
		TeamID:       plan.TeamID.ValueString(),
		FlagID:       plan.FlagID.ValueString(),
		Environments: environments,
	})
	if client.NotFound(err) {
		diags.AddError(
			"Feature Flag not found",
			fmt.Sprintf("Could not find Feature Flag %s %s %s.", plan.TeamID.ValueString(), plan.ProjectID.ValueString(), plan.FlagID.ValueString()),
		)
		return featureFlagConfigModel{}, diags
	}
	if err != nil {
		diags.AddError(
			"Error updating Feature Flag Config",
			fmt.Sprintf(
				"Could not update Feature Flag Config %s %s %s, unexpected error: %s",
				plan.TeamID.ValueString(),
				plan.ProjectID.ValueString(),
				plan.FlagID.ValueString(),
				err,
			),
		)
		return featureFlagConfigModel{}, diags
	}

	result, d := featureFlagConfigFromClient(updated, plan)
	diags.Append(d...)
	return result, diags
}

func featureFlagConfigValidateExistingEnvironments(out client.FeatureFlag) diag.Diagnostics {
	var diags diag.Diagnostics

	for _, name := range []string{"production", "preview", "development"} {
		env, ok := out.Environments[name]
		if !ok {
			continue
		}
		if err := featureFlagEnvironmentShapeError(name, env); err != nil {
			diags.AddError("Unsupported Feature Flag config", err.Error())
		}
	}

	return diags
}

func featureFlagConfigFromClient(out client.FeatureFlag, ref featureFlagConfigModel) (featureFlagConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := featureFlagConfigModel{
		ID:        types.StringValue(out.ID),
		ProjectID: types.StringValue(out.ProjectID),
		TeamID:    ref.TeamID,
		FlagID:    types.StringValue(out.ID),
	}

	production, err := featureFlagEnvironmentFromClient("production", out.Environments["production"])
	if err != nil {
		diags.AddError("Unsupported Feature Flag config", err.Error())
		return model, diags
	}
	preview, err := featureFlagEnvironmentFromClient("preview", out.Environments["preview"])
	if err != nil {
		diags.AddError("Unsupported Feature Flag config", err.Error())
		return model, diags
	}
	development, err := featureFlagEnvironmentFromClient("development", out.Environments["development"])
	if err != nil {
		diags.AddError("Unsupported Feature Flag config", err.Error())
		return model, diags
	}

	model.Production = production
	model.Preview = preview
	model.Development = development
	if model.TeamID.IsNull() {
		model.TeamID = ref.TeamID
	}
	if model.ProjectID.IsNull() {
		model.ProjectID = ref.ProjectID
	}
	if model.FlagID.IsNull() {
		model.FlagID = ref.FlagID
	}
	if model.ID.IsNull() {
		model.ID = ref.ID
	}

	return model, diags
}
