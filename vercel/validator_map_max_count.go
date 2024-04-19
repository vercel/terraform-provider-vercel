package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.Map = validatorMapMaxCount{}

func mapMaxCount(max int) validatorMapMaxCount {
	return validatorMapMaxCount{
		Max: max,
	}
}

type validatorMapMaxCount struct {
	Max int
}

func (v validatorMapMaxCount) Description(ctx context.Context) string {
	return fmt.Sprintf("Map must contain %d or fewer items", v.Max)
}
func (v validatorMapMaxCount) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Map must contain %d or fewer items", v.Max)
}

func (v validatorMapMaxCount) ValidateMap(ctx context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	if len(req.ConfigValue.Elements()) > v.Max {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value provided",
			v.Description(ctx),
		)
		return
	}
}
