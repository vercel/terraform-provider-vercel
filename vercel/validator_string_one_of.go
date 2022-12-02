package vercel

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

func (v validatorStringOneOf) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	if _, ok := v.Items[req.ConfigValue.ValueString()]; !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value provided",
			fmt.Sprintf("Item must be one of %s, got: %s.", strings.Join(v.keys(), ", "), req.ConfigValue.ValueString()),
		)
		return
	}
}
