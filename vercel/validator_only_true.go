package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.Bool = validatorOnlyTrue{}

func onlyTrueValidator(msg string) validatorOnlyTrue {
	return validatorOnlyTrue{msg: msg}
}

type validatorOnlyTrue struct {
	msg string
}

func (v validatorOnlyTrue) Description(ctx context.Context) string {
	return "Value must be true"
}
func (v validatorOnlyTrue) MarkdownDescription(ctx context.Context) string {
	return "Value must be true"
}

func (v validatorOnlyTrue) ValidateBool(ctx context.Context, req validator.BoolRequest, resp *validator.BoolResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	if !req.ConfigValue.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value provided",
			v.msg,
		)
		return
	}
}
