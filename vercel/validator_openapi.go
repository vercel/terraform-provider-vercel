package vercel

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func validateWebhookEvent() validatorOpenAPIEnum {
	return validatorOpenAPIEnum{source: providerOpenAPI, name: "webhook event"}
}

type validatorOpenAPIEnum struct {
	source *openAPIEnums
	name   string
}

func (v validatorOpenAPIEnum) Description(_ context.Context) string {
	return fmt.Sprintf("The %s must be supported by Vercel's OpenAPI schema.", v.name)
}

func (v validatorOpenAPIEnum) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v validatorOpenAPIEnum) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	values, err := v.source.load(ctx)
	if err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Unable to validate attribute", err.Error())
		return
	}

	if !slices.Contains(values[v.name], req.ConfigValue.ValueString()) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Vercel "+v.name,
			fmt.Sprintf("The %s %q is not supported on Vercel. Must be one of %s.", v.name, req.ConfigValue.ValueString(), strings.Join(values[v.name], ", ")))
	}
}
