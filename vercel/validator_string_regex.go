package vercel

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func stringRegex(re *regexp.Regexp, errorMessage string) validatorStringRegex {
	return validatorStringRegex{
		Re:           re,
		ErrorMessage: errorMessage,
	}
}

type validatorStringRegex struct {
	Re           *regexp.Regexp
	ErrorMessage string
}

func (v validatorStringRegex) Description(ctx context.Context) string {
	return v.ErrorMessage
}
func (v validatorStringRegex) MarkdownDescription(ctx context.Context) string {
	return v.ErrorMessage
}

func (v validatorStringRegex) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}
	ok := v.Re.MatchString(req.ConfigValue.ValueString())
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value provided",
			v.ErrorMessage,
		)
		return
	}
}
