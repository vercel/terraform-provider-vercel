package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func mapItemsMinCount(minCount int) validatorMapItemsMinCount {
	return validatorMapItemsMinCount{
		Min: minCount,
	}
}

type validatorMapItemsMinCount struct {
	Min int
}

func (v validatorMapItemsMinCount) Description(ctx context.Context) string {
	return fmt.Sprintf("Map must contain %d or more item(s)", v.Min)
}
func (v validatorMapItemsMinCount) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Map must contain %d or more item(s)", v.Min)
}

func (v validatorMapItemsMinCount) ValidateMap(ctx context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	count := len(req.ConfigValue.Elements())
	if count < v.Min {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value provided",
			fmt.Sprintf(
				"Map must contain at least %d items, got: %d.",
				v.Min,
				count,
			),
		)
		return
	}
}
