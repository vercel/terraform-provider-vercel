package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func int64LessThan(val int64) validatorInt64LessThan {
	return validatorInt64LessThan{
		Max: val,
	}
}

type validatorInt64LessThan struct {
	Max int64
}

func (v validatorInt64LessThan) Description(ctx context.Context) string {
	return fmt.Sprintf("Value must be less than %d", v.Max)
}
func (v validatorInt64LessThan) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Value must be less than `%d`", v.Max)
}

func (v validatorInt64LessThan) Validate(ctx context.Context, req tfsdk.ValidateAttributeRequest, resp *tfsdk.ValidateAttributeResponse) {
	var item types.Int64
	diags := tfsdk.ValueAs(ctx, req.AttributeConfig, &item)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	if item.Unknown || item.Null {
		return
	}

	if item.Value > v.Max {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Invalid value provided",
			fmt.Sprintf("Value must be less than %d, got: %d.", v.Max, item.Value),
		)
		return
	}
}
