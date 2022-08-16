package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

func validateFramework() validatorFramework {
	return validatorFramework{}
}

type validatorFramework struct {
	frameworks []string
}

func (v validatorFramework) Description(ctx context.Context) string {
	if v.frameworks == nil {
		return "The framework provided is not supported on Vercel"
	}
	return stringOneOf(v.frameworks...).Description(ctx)
}

func (v validatorFramework) MarkdownDescription(ctx context.Context) string {
	if v.frameworks == nil {
		return "The framework provided is not supported on Vercel"
	}
	return stringOneOf(v.frameworks...).MarkdownDescription(ctx)
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
		v.frameworks = append(v.frameworks, fw.Slug)
	}

	stringOneOf(v.frameworks...).Validate(ctx, req, resp)
}
