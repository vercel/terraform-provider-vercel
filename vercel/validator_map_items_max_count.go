package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func mapItemsMaxCount(minCount int) validatorMapItemsMaxCount {
	return validatorMapItemsMaxCount{
		Max: minCount,
	}
}

type validatorMapItemsMaxCount struct {
	Max int
}

func (v validatorMapItemsMaxCount) Description(ctx context.Context) string {
	return fmt.Sprintf("Map must contain %d or more item(s)", v.Max)
}
func (v validatorMapItemsMaxCount) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Map must contain %d or more item(s)", v.Max)
}

func (v validatorMapItemsMaxCount) ValidateMap(ctx context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	count := len(req.ConfigValue.Elements())
	if count > v.Max {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value provided",
			fmt.Sprintf(
				"Map must contain no more than %d items, got: %d.",
				v.Max,
				count,
			),
		)
		return
	}
}
