package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func setContainsOnly(ctx context.Context, set types.Set, want string) (bool, diag.Diagnostics) {
	if set.IsNull() || set.IsUnknown() {
		return false, nil
	}

	var values []string
	diags := set.ElementsAs(ctx, &values, true)
	if diags.HasError() {
		return false, diags
	}

	return len(values) == 1 && values[0] == want, nil
}

func setIsUnset(ctx context.Context, set types.Set) (bool, diag.Diagnostics) {
	if set.IsNull() {
		return true, nil
	}
	if set.IsUnknown() {
		return false, nil
	}

	var values []string
	diags := set.ElementsAs(ctx, &values, true)
	if diags.HasError() {
		return false, diags
	}

	return len(values) == 0, nil
}

func shouldValidateSensitiveEnvironmentVariablePolicy(
	ctx context.Context,
	target types.Set,
	customEnvironmentIDs types.Set,
	targetsAllCustomEnvironments bool,
	explicitlyNonSensitive bool,
	id types.String,
) (bool, diag.Diagnostics) {
	if id.ValueString() != "" || !explicitlyNonSensitive {
		return false, nil
	}

	developmentOnly, diags := setContainsOnly(ctx, target, "development")
	if diags.HasError() {
		return false, diags
	}
	if !developmentOnly {
		return true, nil
	}

	// Team sensitivity policy applies to preview and production, not development-only targets.
	customEnvironmentsUnset, diags := setIsUnset(ctx, customEnvironmentIDs)
	if diags.HasError() {
		return false, diags
	}

	return targetsAllCustomEnvironments || !customEnvironmentsUnset, nil
}
