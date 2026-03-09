package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                = &featureFlagResource{}
	_ resource.ResourceWithConfigure   = &featureFlagResource{}
	_ resource.ResourceWithImportState = &featureFlagResource{}
)

var featureFlagKeyRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,512}$`)

func newFeatureFlagResource() resource.Resource {
	return &featureFlagResource{}
}

type featureFlagResource struct {
	client *client.Client
}

type featureFlagEnvironmentModel struct {
	Enabled           types.Bool   `tfsdk:"enabled"`
	DefaultVariantID  types.String `tfsdk:"default_variant_id"`
	DisabledVariantID types.String `tfsdk:"disabled_variant_id"`
}

type featureFlagVariantModel struct {
	ID          types.String `tfsdk:"id"`
	Label       types.String `tfsdk:"label"`
	Description types.String `tfsdk:"description"`
	ValueString types.String `tfsdk:"value_string"`
	ValueNumber types.Number `tfsdk:"value_number"`
	ValueBool   types.Bool   `tfsdk:"value_bool"`
}

type featureFlagModel struct {
	ID          types.String                `tfsdk:"id"`
	ProjectID   types.String                `tfsdk:"project_id"`
	TeamID      types.String                `tfsdk:"team_id"`
	Key         types.String                `tfsdk:"key"`
	Description types.String                `tfsdk:"description"`
	Kind        types.String                `tfsdk:"kind"`
	Archived    types.Bool                  `tfsdk:"archived"`
	Variant     []featureFlagVariantModel   `tfsdk:"variant"`
	Production  featureFlagEnvironmentModel `tfsdk:"production"`
	Preview     featureFlagEnvironmentModel `tfsdk:"preview"`
	Development featureFlagEnvironmentModel `tfsdk:"development"`
}

func (r *featureFlagResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature_flag"
}

func (r *featureFlagResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func featureFlagEnvironmentSchema(description string) schema.SingleNestedAttribute {
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

func (r *featureFlagResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Feature Flag resource.

This first draft keeps the configuration intentionally explicit: you define the variants up front and choose the default and disabled variant for ` + "`production`" + `, ` + "`preview`" + `, and ` + "`development`" + ` separately.

Advanced flag rules, splits, target overrides, and linked environments are not modeled in this resource yet.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the feature flag.",
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
			"key": schema.StringAttribute{
				Required:      true,
				Description:   "The stable flag key used in your application code.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						featureFlagKeyRegex,
						"Flag keys may only contain letters, numbers, dashes, and underscores.",
					),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "A human-readable description of the flag.",
			},
			"kind": schema.StringAttribute{
				Required:      true,
				Description:   "The type of value this flag returns. Must be one of `boolean`, `string`, or `number`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf("boolean", "string", "number"),
				},
			},
			"archived": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the flag should be archived instead of active.",
			},
			"variant": schema.ListNestedAttribute{
				Required:    true,
				Description: "The variants available for this flag.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:    true,
							Description: "The stable variant identifier referenced by each environment.",
						},
						"label": schema.StringAttribute{
							Optional:    true,
							Description: "A human-readable label for the variant.",
						},
						"description": schema.StringAttribute{
							Optional:    true,
							Description: "A human-readable description for the variant.",
						},
						"value_string": schema.StringAttribute{
							Optional:    true,
							Description: "The string value for this variant. Use this when `kind = \"string\"`.",
						},
						"value_number": schema.NumberAttribute{
							Optional:    true,
							Description: "The numeric value for this variant. Use this when `kind = \"number\"`.",
						},
						"value_bool": schema.BoolAttribute{
							Optional:    true,
							Description: "The boolean value for this variant. Use this when `kind = \"boolean\"`.",
						},
					},
				},
			},
			"production": featureFlagEnvironmentSchema("The production environment behavior for this flag."),
			"preview":    featureFlagEnvironmentSchema("The preview environment behavior for this flag."),
			"development": featureFlagEnvironmentSchema(
				"The development environment behavior for this flag.",
			),
		},
	}
}

func (r *featureFlagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan featureFlagModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.TeamID = types.StringValue(r.client.TeamID(plan.TeamID.ValueString()))

	createReq, diags := featureFlagCreateRequest(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createPayload := createReq
	createPayload.State = ""

	out, err := r.client.CreateFeatureFlag(ctx, createPayload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Feature Flag",
			"Could not create Feature Flag, unexpected error: "+err.Error(),
		)
		return
	}

	if createReq.State == "archived" {
		out, err = r.client.UpdateFeatureFlag(ctx, client.UpdateFeatureFlagRequest{
			ProjectID: plan.ProjectID.ValueString(),
			TeamID:    plan.TeamID.ValueString(),
			FlagID:    out.ID,
			State:     "archived",
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error archiving Feature Flag after creation",
				"Could not archive Feature Flag after creation, unexpected error: "+err.Error(),
			)
			return
		}
	}

	result, diags := featureFlagFromClient(out, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "created feature flag", map[string]any{
		"project_id": result.ProjectID.ValueString(),
		"team_id":    result.TeamID.ValueString(),
		"flag_id":    result.ID.ValueString(),
		"key":        result.Key.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *featureFlagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state featureFlagModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.TeamID = types.StringValue(r.client.TeamID(state.TeamID.ValueString()))

	out, err := r.client.GetFeatureFlag(ctx, client.GetFeatureFlagRequest{
		ProjectID: state.ProjectID.ValueString(),
		TeamID:    state.TeamID.ValueString(),
		FlagID:    state.ID.ValueString(),
	})
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Feature Flag",
			fmt.Sprintf(
				"Could not get Feature Flag %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ProjectID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := featureFlagFromClient(out, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *featureFlagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan featureFlagModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.TeamID = types.StringValue(r.client.TeamID(plan.TeamID.ValueString()))

	updateReq, diags := featureFlagUpdateRequest(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateFeatureFlag(ctx, updateReq)
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Feature Flag",
			fmt.Sprintf(
				"Could not update Feature Flag %s %s %s, unexpected error: %s",
				plan.TeamID.ValueString(),
				plan.ProjectID.ValueString(),
				plan.ID.ValueString(),
				err,
			),
		)
		return
	}

	result, diags := featureFlagFromClient(out, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func (r *featureFlagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state featureFlagModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.TeamID = types.StringValue(r.client.TeamID(state.TeamID.ValueString()))

	if !state.Archived.IsNull() && !state.Archived.ValueBool() {
		_, err := r.client.UpdateFeatureFlag(ctx, client.UpdateFeatureFlagRequest{
			ProjectID: state.ProjectID.ValueString(),
			TeamID:    state.TeamID.ValueString(),
			FlagID:    state.ID.ValueString(),
			State:     "archived",
		})
		if client.NotFound(err) {
			return
		}
		if err != nil {
			resp.Diagnostics.AddError(
				"Error archiving Feature Flag before delete",
				fmt.Sprintf(
					"Could not archive Feature Flag %s %s %s before deleting it, unexpected error: %s",
					state.TeamID.ValueString(),
					state.ProjectID.ValueString(),
					state.ID.ValueString(),
					err,
				),
			)
			return
		}
	}

	err := r.client.DeleteFeatureFlag(ctx, client.DeleteFeatureFlagRequest{
		ProjectID: state.ProjectID.ValueString(),
		TeamID:    state.TeamID.ValueString(),
		FlagID:    state.ID.ValueString(),
	})
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Feature Flag",
			fmt.Sprintf(
				"Could not delete Feature Flag %s %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ProjectID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted feature flag", map[string]any{
		"project_id": state.ProjectID.ValueString(),
		"team_id":    state.TeamID.ValueString(),
		"flag_id":    state.ID.ValueString(),
	})
}

func (r *featureFlagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, projectID, flagID, ok := splitInto2Or3(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing Feature Flag",
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
			"Error importing Feature Flag",
			fmt.Sprintf("Could not get Feature Flag %s %s %s, unexpected error: %s", teamID, projectID, flagID, err),
		)
		return
	}

	result, diags := featureFlagFromClient(out, featureFlagModel{
		ProjectID: types.StringValue(projectID),
		TeamID:    types.StringValue(r.client.TeamID(teamID)),
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}

func featureFlagCreateRequest(plan featureFlagModel) (client.CreateFeatureFlagRequest, diag.Diagnostics) {
	req, diags := featureFlagUpsertRequest(plan)
	req.ProjectID = plan.ProjectID.ValueString()
	req.TeamID = plan.TeamID.ValueString()
	req.Key = plan.Key.ValueString()
	req.Kind = plan.Kind.ValueString()
	return req, diags
}

func featureFlagUpdateRequest(plan featureFlagModel) (client.UpdateFeatureFlagRequest, diag.Diagnostics) {
	req, diags := featureFlagUpsertRequest(plan)
	return client.UpdateFeatureFlagRequest{
		ProjectID:    plan.ProjectID.ValueString(),
		TeamID:       plan.TeamID.ValueString(),
		FlagID:       plan.ID.ValueString(),
		Description:  req.Description,
		State:        req.State,
		Variants:     req.Variants,
		Environments: req.Environments,
	}, diags
}

func featureFlagUpsertRequest(plan featureFlagModel) (client.CreateFeatureFlagRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	req := client.CreateFeatureFlagRequest{
		Description:  plan.Description.ValueString(),
		State:        "active",
		Environments: map[string]client.FeatureFlagEnvironment{},
	}
	if !plan.Archived.IsNull() && plan.Archived.ValueBool() {
		req.State = "archived"
	}

	variantIDs := map[string]struct{}{}
	req.Variants = make([]client.FeatureFlagVariant, 0, len(plan.Variant))
	for i, variant := range plan.Variant {
		out, err := featureFlagVariantToClient(plan.Kind.ValueString(), variant)
		if err != nil {
			diags.AddError(
				"Invalid Feature Flag variant",
				fmt.Sprintf("Variant %d is invalid: %s", i+1, err),
			)
			continue
		}
		if _, ok := variantIDs[out.ID]; ok {
			diags.AddError(
				"Duplicate Feature Flag variant ID",
				fmt.Sprintf("Variant ID %q is defined more than once.", out.ID),
			)
			continue
		}
		variantIDs[out.ID] = struct{}{}
		req.Variants = append(req.Variants, out)
	}

	if plan.Kind.ValueString() == "boolean" && len(req.Variants) != 2 {
		diags.AddError(
			"Invalid boolean Feature Flag variants",
			"Boolean flags must define exactly two variants.",
		)
	}

	environments := map[string]featureFlagEnvironmentModel{
		"production":  plan.Production,
		"preview":     plan.Preview,
		"development": plan.Development,
	}
	for name, env := range environments {
		revision := 0
		if _, ok := variantIDs[env.DefaultVariantID.ValueString()]; !ok {
			diags.AddError(
				"Unknown Feature Flag default variant",
				fmt.Sprintf("%s.default_variant_id references %q, but no variant with that ID exists.", name, env.DefaultVariantID.ValueString()),
			)
		}
		if _, ok := variantIDs[env.DisabledVariantID.ValueString()]; !ok {
			diags.AddError(
				"Unknown Feature Flag disabled variant",
				fmt.Sprintf("%s.disabled_variant_id references %q, but no variant with that ID exists.", name, env.DisabledVariantID.ValueString()),
			)
		}
		req.Environments[name] = client.FeatureFlagEnvironment{
			Active:   env.Enabled.ValueBool(),
			Revision: &revision,
			PausedOutcome: client.FeatureFlagOutcome{
				Type:      "variant",
				VariantID: env.DisabledVariantID.ValueString(),
			},
			Fallthrough: client.FeatureFlagOutcome{
				Type:      "variant",
				VariantID: env.DefaultVariantID.ValueString(),
			},
			Rules: []json.RawMessage{},
			Reuse: &client.FeatureFlagReuse{
				Active:      false,
				Environment: "",
			},
		}
	}

	return req, diags
}

func featureFlagVariantToClient(kind string, variant featureFlagVariantModel) (client.FeatureFlagVariant, error) {
	out := client.FeatureFlagVariant{
		ID:          variant.ID.ValueString(),
		Label:       variant.Label.ValueString(),
		Description: variant.Description.ValueString(),
	}

	setCount := 0
	if !variant.ValueString.IsNull() {
		setCount++
	}
	if !variant.ValueNumber.IsNull() {
		setCount++
	}
	if !variant.ValueBool.IsNull() {
		setCount++
	}
	if setCount != 1 {
		return out, fmt.Errorf("exactly one of value_string, value_number, or value_bool must be set")
	}

	switch kind {
	case "string":
		if variant.ValueString.IsNull() {
			return out, fmt.Errorf("string flags require value_string")
		}
		out.Value = variant.ValueString.ValueString()
	case "number":
		if variant.ValueNumber.IsNull() {
			return out, fmt.Errorf("number flags require value_number")
		}
		floatValue, _ := variant.ValueNumber.ValueBigFloat().Float64()
		out.Value = floatValue
	case "boolean":
		if variant.ValueBool.IsNull() {
			return out, fmt.Errorf("boolean flags require value_bool")
		}
		out.Value = variant.ValueBool.ValueBool()
	default:
		return out, fmt.Errorf("unsupported kind %q", kind)
	}

	return out, nil
}

func featureFlagFromClient(out client.FeatureFlag, ref featureFlagModel) (featureFlagModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := featureFlagModel{
		ID:          types.StringValue(out.ID),
		ProjectID:   types.StringValue(out.ProjectID),
		TeamID:      ref.TeamID,
		Key:         types.StringValue(out.Slug),
		Description: featureFlagOptionalStringValue(out.Description, ref.Description),
		Kind:        types.StringValue(out.Kind),
		Archived:    types.BoolValue(out.State == "archived"),
	}

	priorVariants := map[string]featureFlagVariantModel{}
	for _, variant := range ref.Variant {
		priorVariants[variant.ID.ValueString()] = variant
	}

	var variants = make([]featureFlagVariantModel, 0, len(out.Variants))
	for _, variant := range out.Variants {
		mapped, err := featureFlagVariantFromClient(out.Kind, variant, priorVariants[variant.ID])
		if err != nil {
			diags.AddError(
				"Unsupported Feature Flag variant",
				fmt.Sprintf("Feature flag %q has a variant that cannot be represented by this resource: %s", out.Slug, err),
			)
			return model, diags
		}
		variants = append(variants, mapped)
	}
	model.Variant = variants

	production, err := featureFlagEnvironmentFromClient("production", out.Environments["production"])
	if err != nil {
		diags.AddError("Unsupported Feature Flag environment", err.Error())
		return model, diags
	}
	preview, err := featureFlagEnvironmentFromClient("preview", out.Environments["preview"])
	if err != nil {
		diags.AddError("Unsupported Feature Flag environment", err.Error())
		return model, diags
	}
	development, err := featureFlagEnvironmentFromClient("development", out.Environments["development"])
	if err != nil {
		diags.AddError("Unsupported Feature Flag environment", err.Error())
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

	return model, diags
}

func featureFlagVariantFromClient(kind string, variant client.FeatureFlagVariant, prior featureFlagVariantModel) (featureFlagVariantModel, error) {
	model := featureFlagVariantModel{
		ID:          types.StringValue(variant.ID),
		Label:       featureFlagOptionalStringValue(variant.Label, prior.Label),
		Description: featureFlagOptionalStringValue(variant.Description, prior.Description),
		ValueString: types.StringNull(),
		ValueNumber: types.NumberNull(),
		ValueBool:   types.BoolNull(),
	}

	switch value := variant.Value.(type) {
	case string:
		if kind != "string" {
			return model, fmt.Errorf("received string variant value for %s flag", kind)
		}
		model.ValueString = types.StringValue(value)
	case float64:
		if kind != "number" {
			return model, fmt.Errorf("received numeric variant value for %s flag", kind)
		}
		model.ValueNumber = types.NumberValue(big.NewFloat(value))
	case bool:
		if kind != "boolean" {
			return model, fmt.Errorf("received boolean variant value for %s flag", kind)
		}
		model.ValueBool = types.BoolValue(value)
	default:
		return model, fmt.Errorf("unsupported variant value type %T", variant.Value)
	}

	return model, nil
}

func featureFlagEnvironmentFromClient(name string, env client.FeatureFlagEnvironment) (featureFlagEnvironmentModel, error) {
	if env.Reuse != nil && env.Reuse.Active {
		return featureFlagEnvironmentModel{}, fmt.Errorf("%s uses a linked environment, which this resource does not model yet", name)
	}
	if len(env.Rules) > 0 {
		return featureFlagEnvironmentModel{}, fmt.Errorf("%s defines rules, which this resource does not model yet", name)
	}
	if len(env.Targets) > 0 {
		return featureFlagEnvironmentModel{}, fmt.Errorf("%s defines target overrides, which this resource does not model yet", name)
	}
	if env.Fallthrough.Type != "variant" {
		return featureFlagEnvironmentModel{}, fmt.Errorf("%s uses a non-variant fallthrough outcome, which this resource does not model yet", name)
	}
	if env.PausedOutcome.Type != "variant" {
		return featureFlagEnvironmentModel{}, fmt.Errorf("%s uses a non-variant paused outcome, which this resource does not model yet", name)
	}

	return featureFlagEnvironmentModel{
		Enabled:           types.BoolValue(env.Active),
		DefaultVariantID:  types.StringValue(env.Fallthrough.VariantID),
		DisabledVariantID: types.StringValue(env.PausedOutcome.VariantID),
	}, nil
}

func featureFlagOptionalStringValue(value string, prior types.String) types.String {
	if value == "" {
		if !prior.IsNull() {
			return prior
		}
		return types.StringNull()
	}
	return types.StringValue(value)
}
