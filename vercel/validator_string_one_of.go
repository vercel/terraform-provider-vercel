package vercel

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stringOneOf(items ...string) validatorStringOneOf {
	itemMap := map[string]struct{}{}
	for _, i := range items {
		itemMap[i] = struct{}{}
	}
	return validatorStringOneOf{
		Items: itemMap,
	}
}

type validatorStringOneOf struct {
	Items map[string]struct{}
}

func (v validatorStringOneOf) keys() (out []string) {
	for k := range v.Items {
		out = append(out, k)
	}
	return
}

func (v validatorStringOneOf) Description(ctx context.Context) string {
	return fmt.Sprintf("Item must be one of %s", strings.Join(v.keys(), " "))
}
func (v validatorStringOneOf) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Item must be one of `%s`", strings.Join(v.keys(), "` `"))
}

func (v validatorStringOneOf) Validate(ctx context.Context, req tfsdk.ValidateAttributeRequest, resp *tfsdk.ValidateAttributeResponse) {
	var item types.String
	diags := tfsdk.ValueAs(ctx, req.AttributeConfig, &item)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	if item.IsUnknown() || item.IsNull() {
		return
	}

	if _, ok := v.Items[item.ValueString()]; !ok {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Invalid value provided",
			fmt.Sprintf("Item must be one of %s, got: %s.", strings.Join(v.keys(), ", "), item.ValueString()),
		)
		return
	}
}
