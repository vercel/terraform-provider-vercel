package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func setMinSize(minSize int) validatorSetMinSize {
	return validatorSetMinSize{
		Min: minSize,
	}
}

type validatorSetMinSize struct {
	Min int
}

func (v validatorSetMinSize) Description(ctx context.Context) string {
	return fmt.Sprintf("set must contain at least %d item", v.Min)
}
func (v validatorSetMinSize) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("set must contain at least `%d` item", v.Min)
}

func (v validatorSetMinSize) Validate(ctx context.Context, req tfsdk.ValidateAttributeRequest, resp *tfsdk.ValidateAttributeResponse) {
	var set types.Set
	diags := tfsdk.ValueAs(ctx, req.AttributeConfig, &set)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	if set.Unknown || set.Null {
		return
	}

	if len(set.Elems) < v.Min {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Invalid Set Length",
			fmt.Sprintf("Set must contain at least %d items, got: %d.", v.Min, len(set.Elems)),
		)
		return
	}
}
