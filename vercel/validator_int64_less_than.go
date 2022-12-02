package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

func (v validatorInt64LessThan) ValidateInt64(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	if req.ConfigValue.ValueInt64() > v.Max {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value provided",
			fmt.Sprintf("Value must be less than %d, got: %d.", v.Max, req.ConfigValue.ValueInt64()),
		)
		return
	}
}
