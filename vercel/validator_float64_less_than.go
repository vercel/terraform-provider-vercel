package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func float64LessThan(val float64) validatorFloat64LessThan {
	return validatorFloat64LessThan{
		Max: val,
	}
}

type validatorFloat64LessThan struct {
	Max float64
}

func (v validatorFloat64LessThan) Description(ctx context.Context) string {
	return fmt.Sprintf("Value must be less than %.2f", v.Max)
}
func (v validatorFloat64LessThan) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Value must be less than `%.2f`", v.Max)
}

func (v validatorFloat64LessThan) ValidateFloat64(ctx context.Context, req validator.Float64Request, resp *validator.Float64Response) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	if req.ConfigValue.ValueFloat64() > v.Max {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value provided",
			fmt.Sprintf("Value must be less than %.2f, got: %.2f.", v.Max, req.ConfigValue.ValueFloat64()),
		)
		return
	}
}
