package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func mapItemsMinCount(minCount int) validatorMapItemsMinCount {
	return validatorMapItemsMinCount{
		Min: minCount,
	}
}

type validatorMapItemsMinCount struct {
	Max int
	Min int
}

func (v validatorMapItemsMinCount) Description(ctx context.Context) string {
	return fmt.Sprintf("Map must contain at least %d item(s)", v.Min)
}
func (v validatorMapItemsMinCount) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Map must contain at least `%d` item(s)", v.Min)
}

func (v validatorMapItemsMinCount) Validate(ctx context.Context, req tfsdk.ValidateAttributeRequest, resp *tfsdk.ValidateAttributeResponse) {
	var val types.Map
	diags := tfsdk.ValueAs(ctx, req.AttributeConfig, &val)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	if val.IsNull() || val.IsUnknown() {
		return
	}
	count := len(val.Elements())
	if count < v.Min {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
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
