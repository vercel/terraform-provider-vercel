package vercel

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = validatorJSON{}

func validateJSON() validatorJSON {
	return validatorJSON{}
}

type validatorJSON struct {
}

func (v validatorJSON) Description(ctx context.Context) string {
	return "Value must be valid JSON"
}
func (v validatorJSON) MarkdownDescription(ctx context.Context) string {
	return "Value must be valid JSON"
}

func (v validatorJSON) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	var i any
	if err := json.Unmarshal([]byte(req.ConfigValue.ValueString()), &i); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value provided",
			fmt.Sprintf("Value must be a valid JSON document, but it could not be parsed: %s.", err),
		)
		return
	}
}
