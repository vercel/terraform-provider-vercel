package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func int64GreaterThan(val int64) validatorInt64GreaterThan {
	return validatorInt64GreaterThan{
		Min: val,
	}
}

type validatorInt64GreaterThan struct {
	Min int64
}

func (v validatorInt64GreaterThan) Description(ctx context.Context) string {
	return fmt.Sprintf("Value must be greater than %d", v.Min)
}
func (v validatorInt64GreaterThan) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Value must be greater than `%d`", v.Min)
}

func (v validatorInt64GreaterThan) Validate(ctx context.Context, req tfsdk.ValidateAttributeRequest, resp *tfsdk.ValidateAttributeResponse) {
	var item types.Int64
	diags := tfsdk.ValueAs(ctx, req.AttributeConfig, &item)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	if item.Unknown || item.Null {
		return
	}

	if item.Value < v.Min {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Invalid value provided",
			fmt.Sprintf("Value must be greater than %d, got: %d.", v.Min, item.Value),
		)
		return
	}
}
