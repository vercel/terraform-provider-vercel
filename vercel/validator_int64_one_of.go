package vercel

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func int64OneOf(items ...int64) validatorInt64OneOf {
	itemMap := map[int64]struct{}{}
	for _, i := range items {
		itemMap[i] = struct{}{}
	}
	return validatorInt64OneOf{
		Items: itemMap,
	}
}

type validatorInt64OneOf struct {
	Items map[int64]struct{}
}

func (v validatorInt64OneOf) keys() (out []string) {
	for k := range v.Items {
		out = append(out, strconv.Itoa(int(k)))
	}
	return
}

func (v validatorInt64OneOf) Description(ctx context.Context) string {
	return fmt.Sprintf("Item must be one of %s", strings.Join(v.keys(), " "))
}
func (v validatorInt64OneOf) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Item must be one of `%s`", strings.Join(v.keys(), "` `"))
}

func (v validatorInt64OneOf) ValidateInt64(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	if _, ok := v.Items[req.ConfigValue.ValueInt64()]; !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value provided",
			fmt.Sprintf("Item must be one of %s, got: %d.", strings.Join(v.keys(), " "), req.ConfigValue.ValueInt64()),
		)
		return
	}
}
