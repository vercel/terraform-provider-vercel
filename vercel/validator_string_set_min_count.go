package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func stringSetMinCount(min int) validatorStringSetMinCount {
	return validatorStringSetMinCount{
		Min: min,
	}
}

type validatorStringSetMinCount struct {
	Min int
}

func (v validatorStringSetMinCount) Description(ctx context.Context) string {
	return fmt.Sprintf("Set must contain at least %d items", v.Min)
}
func (v validatorStringSetMinCount) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Set must contain at least %d items", v.Min)
}

func (v validatorStringSetMinCount) ValidateSet(ctx context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	if len(req.ConfigValue.Elements()) < v.Min {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value provided",
			v.Description(ctx),
		)
		return
	}
}
