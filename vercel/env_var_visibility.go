package vercel

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func environmentVariableVisibilitySchemaAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Description: "Controls how the environment variable is categorized: `config` (configuration values) or `secret` (secret values). When omitted, visibility is inferred from `sensitive` for backwards compatibility and is not sent to the API.",
		Optional:    true,
		Computed:    true,
		Validators: []validator.String{
			stringvalidator.OneOf("config", "secret"),
		},
	}
}

func resolveEnvVarTypeAndVisibility(sensitive types.Bool, visibility types.String) (envType string, visibilityOut *string, diags diag.Diagnostics) {
	visibilitySet := !visibility.IsNull() && !visibility.IsUnknown()

	if visibilitySet {
		v := visibility.ValueString()
		switch v {
		case "secret":
			if isExplicitlyNonSensitive(sensitive) {
				diags.AddError(
					"Invalid environment variable configuration",
					"`visibility` must be `config` when `sensitive` is false.",
				)
				return "", nil, diags
			}
			envType = "sensitive"
			visibilityOut = &v
		case "config":
			if !sensitive.IsNull() && !sensitive.IsUnknown() && sensitive.ValueBool() {
				diags.AddError(
					"Invalid environment variable configuration",
					"`visibility` must be `secret` when `sensitive` is true.",
				)
				return "", nil, diags
			}
			envType = "encrypted"
			visibilityOut = &v
		}
		return envType, visibilityOut, diags
	}

	if isExplicitlyNonSensitive(sensitive) {
		return "encrypted", nil, diags
	}
	return "sensitive", nil, diags
}

func envVarVisibilityFromResponse(envType string, apiVisibility *string) types.String {
	if apiVisibility != nil && *apiVisibility != "" {
		return types.StringValue(*apiVisibility)
	}
	if envType == "sensitive" {
		return types.StringValue("secret")
	}
	return types.StringNull()
}

func isExplicitlyNonSensitive(sensitive types.Bool) bool {
	return !sensitive.IsNull() && !sensitive.IsUnknown() && !sensitive.ValueBool()
}
