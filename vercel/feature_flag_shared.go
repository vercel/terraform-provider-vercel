package vercel

import (
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

var featureFlagKeyRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,512}$`)

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

func featureFlagVariantsToClient(kind string, variants []featureFlagVariantModel) ([]client.FeatureFlagVariant, map[string]struct{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	variantIDs := map[string]struct{}{}
	out := make([]client.FeatureFlagVariant, 0, len(variants))
	for i, variant := range variants {
		mapped, err := featureFlagVariantToClient(kind, variant)
		if err != nil {
			diags.AddError(
				"Invalid Feature Flag variant",
				fmt.Sprintf("Variant %d is invalid: %s", i+1, err),
			)
			continue
		}
		if _, ok := variantIDs[mapped.ID]; ok {
			diags.AddError(
				"Duplicate Feature Flag variant ID",
				fmt.Sprintf("Variant ID %q is defined more than once.", mapped.ID),
			)
			continue
		}
		variantIDs[mapped.ID] = struct{}{}
		out = append(out, mapped)
	}

	if len(out) == 0 {
		diags.AddError(
			"Invalid Feature Flag variants",
			"At least one valid variant must be defined.",
		)
	}
	if kind == "boolean" && len(out) != 2 {
		diags.AddError(
			"Invalid boolean Feature Flag variants",
			"Boolean flags must define exactly two variants.",
		)
	}

	return out, variantIDs, diags
}

func featureFlagVariantIDsFromClient(variants []client.FeatureFlagVariant) map[string]struct{} {
	out := make(map[string]struct{}, len(variants))
	for _, variant := range variants {
		out[variant.ID] = struct{}{}
	}
	return out
}

func featureFlagEnvironmentsFromModel(environments map[string]featureFlagEnvironmentModel, variantIDs map[string]struct{}) (map[string]client.FeatureFlagEnvironment, diag.Diagnostics) {
	var diags diag.Diagnostics

	out := make(map[string]client.FeatureFlagEnvironment, len(environments))
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

		out[name] = client.FeatureFlagEnvironment{
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

	return out, diags
}

func featureFlagBootstrapVariantID(kind string, variants []client.FeatureFlagVariant) (string, error) {
	switch kind {
	case "boolean":
		for _, variant := range variants {
			value, ok := variant.Value.(bool)
			if ok && !value {
				return variant.ID, nil
			}
		}
		return "", fmt.Errorf("boolean feature flag definitions must include a variant with value_bool = false because Vercel requires paused environments on create")
	case "string", "number":
		for _, variant := range variants {
			if variant.ID == "control" {
				return variant.ID, nil
			}
		}
		return "", fmt.Errorf("%s feature flag definitions must include a variant with id %q because Vercel requires paused environments on create", kind, "control")
	default:
		return "", fmt.Errorf("unsupported kind %q", kind)
	}
}

func featureFlagBootstrapEnvironments(defaultVariantID string) map[string]client.FeatureFlagEnvironment {
	out := make(map[string]client.FeatureFlagEnvironment, 3)

	for _, name := range []string{"production", "preview", "development"} {
		revision := 0
		out[name] = client.FeatureFlagEnvironment{
			Active:   false,
			Revision: &revision,
			PausedOutcome: client.FeatureFlagOutcome{
				Type:      "variant",
				VariantID: defaultVariantID,
			},
			Fallthrough: client.FeatureFlagOutcome{
				Type:      "variant",
				VariantID: defaultVariantID,
			},
			Rules: []json.RawMessage{},
			Reuse: &client.FeatureFlagReuse{
				Active:      false,
				Environment: "",
			},
		}
	}

	return out
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

	variants := make([]featureFlagVariantModel, 0, len(out.Variants))
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
	if err := featureFlagEnvironmentShapeError(name, env); err != nil {
		return featureFlagEnvironmentModel{}, err
	}

	return featureFlagEnvironmentModel{
		Enabled:           types.BoolValue(env.Active),
		DefaultVariantID:  types.StringValue(env.Fallthrough.VariantID),
		DisabledVariantID: types.StringValue(env.PausedOutcome.VariantID),
	}, nil
}

func featureFlagEnvironmentShapeError(name string, env client.FeatureFlagEnvironment) error {
	if env.Reuse != nil && env.Reuse.Active {
		return fmt.Errorf("%s uses a linked environment, which this resource does not model yet", name)
	}
	if len(env.Rules) > 0 {
		return fmt.Errorf("%s defines rules, which this resource does not model yet", name)
	}
	if len(env.Targets) > 0 {
		return fmt.Errorf("%s defines target overrides, which this resource does not model yet", name)
	}
	if env.Fallthrough.Type != "variant" {
		return fmt.Errorf("%s uses a non-variant fallthrough outcome, which this resource does not model yet", name)
	}
	if env.PausedOutcome.Type != "variant" {
		return fmt.Errorf("%s uses a non-variant paused outcome, which this resource does not model yet", name)
	}

	return nil
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
