package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

func (v validatorFramework) Validate(ctx context.Context, req tfsdk.ValidateAttributeRequest, resp *tfsdk.ValidateAttributeResponse) {
	apires, err := http.Get("https://api-frameworks.zeit.sh/")
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Unable to validate attribute",
			fmt.Sprintf("Unable to retrieve Vercel frameworks: unexpected error: %s", err),
		)
		return
	}
	if apires.StatusCode != 200 {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Unable to validate attribute",
			fmt.Sprintf("Unable to retrieve Vercel frameworks: unexpected status code: %d", apires.StatusCode),
		)
		return
	}

	defer apires.Body.Close()
	responseBody, err := io.ReadAll(apires.Body)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
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
			req.AttributePath,
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

	var item types.String
	diags := tfsdk.ValueAs(ctx, req.AttributeConfig, &item)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	if item.Unknown || item.Null {
		return
	}

	if _, ok := v.frameworks[item.Value]; !ok {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Invalid Framework",
			fmt.Sprintf("The framework %s is not supported on Vercel. Must be one of %s.", item.Value, strings.Join(keys(v.frameworks), ", ")),
		)
		return
	}
}
