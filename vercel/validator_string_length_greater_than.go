package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = validatorStringLengthGreaterThan{}

func stringLengthGreaterThan(min int) validatorStringLengthGreaterThan {
	return validatorStringLengthGreaterThan{
		Min: min,
	}
}

type validatorStringLengthGreaterThan struct {
	Min int
}

func (v validatorStringLengthGreaterThan) Description(ctx context.Context) string {
	return fmt.Sprintf("String length must be equal to or greater than %d", v.Min)
}

func (v validatorStringLengthGreaterThan) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("String length must be equal to or greater than %d", v.Min)
}

func (v validatorStringLengthGreaterThan) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}
	strLen := len(req.ConfigValue.ValueString())
	if strLen < v.Min {
		resp.Diagnostics.AddError(
			"Invalid value provided",
			fmt.Sprintf("String length must be greater than %d, got: %d.", v.Min, strLen),
		)
		return
	}
}
