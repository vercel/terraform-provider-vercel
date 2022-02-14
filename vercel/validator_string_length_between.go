package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stringLengthBetween(minLength int, maxLength int) validatorStringLengthBetween {
	return validatorStringLengthBetween{
		Max: maxLength,
		Min: minLength,
	}
}

type validatorStringLengthBetween struct {
	Max int
	Min int
}

func (v validatorStringLengthBetween) Description(ctx context.Context) string {
	return fmt.Sprintf("String length must be between %d and %d", v.Min, v.Max)
}
func (v validatorStringLengthBetween) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("String length must be between `%d` and `%d`", v.Min, v.Max)
}

func (v validatorStringLengthBetween) Validate(ctx context.Context, req tfsdk.ValidateAttributeRequest, resp *tfsdk.ValidateAttributeResponse) {
	var str types.String
	diags := tfsdk.ValueAs(ctx, req.AttributeConfig, &str)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	if str.Unknown || str.Null {
		return
	}
	strLen := len(str.Value)
	if strLen < v.Min || strLen > v.Max {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Invalid value provided",
			fmt.Sprintf("String length must be between %d and %d, got: %d.", v.Min, v.Max, strLen),
		)
		return
	}
}
