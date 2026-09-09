package vercel

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = validatorStringIsTrimmed{}

func validateStringIsTrimmed() validatorStringIsTrimmed {
	return validatorStringIsTrimmed{}
}

type validatorStringIsTrimmed struct{}

func (validatorStringIsTrimmed) Description(context.Context) string {
	return "Value must not have leading or trailing whitespace"
}

func (validatorStringIsTrimmed) MarkdownDescription(context.Context) string {
	return "Value must not have leading or trailing whitespace"
}

func (validatorStringIsTrimmed) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if value := req.ConfigValue.ValueString(); value != strings.TrimSpace(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid untrimmed value",
			"Value must not have leading or trailing whitespace because the Vercel API trims it.",
		)
	}
}
