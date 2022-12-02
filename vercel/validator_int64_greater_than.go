package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

func (v validatorInt64GreaterThan) ValidateInt64(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	if req.ConfigValue.ValueInt64() < v.Min {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value provided",
			fmt.Sprintf("Value must be greater than %d, got: %d.", v.Min, req.ConfigValue.ValueInt64()),
		)
		return
	}
}
