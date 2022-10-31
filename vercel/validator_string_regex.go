package vercel

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

func (v validatorStringRegex) Validate(ctx context.Context, req tfsdk.ValidateAttributeRequest, resp *tfsdk.ValidateAttributeResponse) {
	var str types.String
	diags := tfsdk.ValueAs(ctx, req.AttributeConfig, &str)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	if str.IsUnknown() || str.IsNull() {
		return
	}
	ok := v.Re.MatchString(str.ValueString())
	if !ok {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Invalid value provided",
			v.ErrorMessage,
		)
		return
	}
}
