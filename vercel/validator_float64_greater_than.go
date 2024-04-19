package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func float64GreaterThan(val float64) validatorFloat64GreaterThan {
	return validatorFloat64GreaterThan{
		Min: val,
	}
}

type validatorFloat64GreaterThan struct {
	Min float64
}

func (v validatorFloat64GreaterThan) Description(ctx context.Context) string {
	return fmt.Sprintf("Value must be equal to or greater than %.2f", v.Min)
}
func (v validatorFloat64GreaterThan) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Value must be equal to or greater than `%.2f`", v.Min)
}

func (v validatorFloat64GreaterThan) ValidateFloat64(ctx context.Context, req validator.Float64Request, resp *validator.Float64Response) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	if req.ConfigValue.ValueFloat64() < v.Min {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value provided",
			fmt.Sprintf("Value must be greater than %.2f, got: %.2f.", v.Min, req.ConfigValue.ValueFloat64()),
		)
		return
	}
}
