package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func validateFramework() validatorFramework {
	return validatorFramework{}
}

type validatorFramework struct {
	frameworks map[string]struct{}
}

func (v validatorFramework) Description(ctx context.Context) string {
	if v.frameworks == nil {
		return "The framework provided is not supported on Vercel"
	}
	return fmt.Sprintf("The framework provided is not supported on Vercel. Must be one of %s.", strings.Join(keys(v.frameworks), ", "))
}

func (v validatorFramework) MarkdownDescription(ctx context.Context) string {
	if v.frameworks == nil {
		return "The framework provided is not supported on Vercel"
	}
	return fmt.Sprintf("The framework provided is not supported on Vercel. Must be one of `%s`.", strings.Join(keys(v.frameworks), "`, `"))
}

func (v validatorFramework) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	apires, err := http.Get("https://api-frameworks.zeit.sh/")
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Unable to validate attribute",
			fmt.Sprintf("Unable to retrieve Vercel frameworks: unexpected error: %s", err),
		)
		return
	}
	if apires.StatusCode != 200 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Unable to validate attribute",
			fmt.Sprintf("Unable to retrieve Vercel frameworks: unexpected status code: %d", apires.StatusCode),
		)
		return
	}

	defer apires.Body.Close()
	responseBody, err := io.ReadAll(apires.Body)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Unable to validate attribute",
			fmt.Sprintf("Unable to retrieve Vercel frameworks: error reading response body: %s", err),
		)
		return
	}
	var fwList []struct {
		Slug string `json:"slug"`
	}
	err = json.Unmarshal(responseBody, &fwList)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Unable to validate attribute",
			fmt.Sprintf("Unable to retrieve Vercel frameworks: error parsing frameworks response: %s", err),
		)
		return
	}
	for _, fw := range fwList {
		if v.frameworks == nil {
			v.frameworks = map[string]struct{}{}
		}
		v.frameworks[fw.Slug] = struct{}{}
	}

	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	if _, ok := v.frameworks[req.ConfigValue.ValueString()]; !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Framework",
			fmt.Sprintf("The framework %s is not supported on Vercel. Must be one of %s.", req.ConfigValue.ValueString(), strings.Join(keys(v.frameworks), ", ")),
		)
		return
	}
}
