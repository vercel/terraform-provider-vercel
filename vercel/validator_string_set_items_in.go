package vercel

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stringSetItemsIn(items ...string) validatorStringSetItemsIn {
	itemMap := map[string]struct{}{}
	for _, i := range items {
		itemMap[i] = struct{}{}
	}
	return validatorStringSetItemsIn{
		Items: itemMap,
	}
}

type validatorStringSetItemsIn struct {
	Items map[string]struct{}
}

func (v validatorStringSetItemsIn) keys() (out []string) {
	for k := range v.Items {
		out = append(out, k)
	}
	return
}

func (v validatorStringSetItemsIn) Description(ctx context.Context) string {
	return fmt.Sprintf("Set item must be one of %s", strings.Join(v.keys(), ", "))
}
func (v validatorStringSetItemsIn) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Set item must be one of `%s`", strings.Join(v.keys(), ",` `"))
}

func (v validatorStringSetItemsIn) ValidateSet(ctx context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	for _, i := range req.ConfigValue.Elements() {
		var item types.String
		if item.IsUnknown() || item.IsNull() {
			continue
		}
		diags := tfsdk.ValueAs(ctx, i, &item)
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			return
		}
		if _, ok := v.Items[item.ValueString()]; !ok {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid value provided",
				fmt.Sprintf("%s, got %s", v.Description(ctx), item.ValueString()),
			)
			return
		}
	}
}
