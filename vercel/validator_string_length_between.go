package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

func (v validatorStringLengthBetween) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}
	strLen := len(req.ConfigValue.ValueString())
	if strLen < v.Min || strLen > v.Max {
		resp.Diagnostics.AddError(
			"Invalid value provided",
			fmt.Sprintf("String length must be between %d and %d, got: %d.", v.Min, v.Max, strLen),
		)
		return
	}
}
