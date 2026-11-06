package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

// settingsUpdate records settings omitted from a PATCH because their planned
// values are unchanged. Keep them consistent with the plan until the next Read,
// without overwriting concurrent changes through the API.
type settingsUpdate struct {
	retained []path.Path
}

func (u *settingsUpdate) changed(p path.Path, plan, state attr.Value) bool {
	if plan.Equal(state) {
		u.retained = append(u.retained, p)
		return false
	}
	return true
}

func (u *settingsUpdate) retainPlannedValues(ctx context.Context, plan tfsdk.Plan, state *tfsdk.State) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, p := range u.retained {
		var value attr.Value
		diags.Append(plan.GetAttribute(ctx, p, &value)...)
		if diags.HasError() {
			return diags
		}
		diags.Append(state.SetAttribute(ctx, p, value)...)
		if diags.HasError() {
			return diags
		}
	}
	return diags
}
