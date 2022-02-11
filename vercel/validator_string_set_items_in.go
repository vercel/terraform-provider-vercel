package vercel

import (
	"context"
	"fmt"
	"strings"

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
	return fmt.Sprintf("set item must be one of %s", strings.Join(v.keys(), " "))
}
func (v validatorStringSetItemsIn) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("set item must be one of `%s`", strings.Join(v.keys(), "` `"))
}

func (v validatorStringSetItemsIn) Validate(ctx context.Context, req tfsdk.ValidateAttributeRequest, resp *tfsdk.ValidateAttributeResponse) {
	var set types.Set
	diags := tfsdk.ValueAs(ctx, req.AttributeConfig, &set)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	if set.Unknown || set.Null {
		return
	}

	for _, i := range set.Elems {
		var item types.String
		diags := tfsdk.ValueAs(ctx, i, &item)
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			return
		}
		if set.Unknown || set.Null {
			return
		}
		if _, ok := v.Items[item.Value]; !ok {
			resp.Diagnostics.AddAttributeError(
				req.AttributePath,
				"Invalid value provided",
				fmt.Sprintf("Set item must be one of %s, got: %s.", strings.Join(v.keys(), " "), item.Value),
			)
			return
		}
	}
}
