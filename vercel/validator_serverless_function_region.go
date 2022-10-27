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

func validateServerlessFunctionRegion() validatorServerlessFunctionRegion {
	return validatorServerlessFunctionRegion{}
}

type validatorServerlessFunctionRegion struct {
	regions map[string]struct{}
}

func (v validatorServerlessFunctionRegion) Description(ctx context.Context) string {
	if v.regions == nil {
		return "The serverless function region provided is not supported on Vercel"
	}
	return fmt.Sprintf("The serverless function region provided is not supported on Vercel. Must be one of %s.", strings.Join(keys(v.regions), ", "))
}

func (v validatorServerlessFunctionRegion) MarkdownDescription(ctx context.Context) string {
	if v.regions == nil {
		return "The serverless function region provided is not supported on Vercel"
	}
	return fmt.Sprintf("The serverless function region provided is not supported on Vercel. Must be one of `%s`.", strings.Join(keys(v.regions), "`, `"))
}

func contains(items []string, i string) bool {
	for _, j := range items {
		if j == i {
			return true
		}
	}
	return false
}

func keys(v map[string]struct{}) (out []string) {
	for k := range v {
		out = append(out, k)
	}
	return
}

func (v validatorServerlessFunctionRegion) Validate(ctx context.Context, req tfsdk.ValidateAttributeRequest, resp *tfsdk.ValidateAttributeResponse) {
	apires, err := http.Get("https://dcs.vercel-infra.com")
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Unable to validate attribute",
			fmt.Sprintf("Unable to retrieve Vercel serverless function regions: unexpected error: %s", err),
		)
		return
	}
	if apires.StatusCode != 200 {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Unable to validate attribute",
			fmt.Sprintf("Unable to retrieve Vercel serverless function regions: unexpected status code: %d", apires.StatusCode),
		)
		return
	}

	defer apires.Body.Close()
	responseBody, err := io.ReadAll(apires.Body)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Unable to validate attribute",
			fmt.Sprintf("Unable to retrieve Vercel serverless function regions: error reading response body: %s", err),
		)
		return
	}

	var regions map[string]struct {
		Caps []string `json:"caps"`
	}
	err = json.Unmarshal(responseBody, &regions)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Unable to validate attribute",
			fmt.Sprintf("Unable to retrieve Vercel serverless function regions: error parsing serverless function regions response: %s", err),
		)
		return
	}

	for region, regionInfo := range regions {
		if contains(regionInfo.Caps, "V2_DEPLOYMENT_CREATE") {
			if v.regions == nil {
				v.regions = map[string]struct{}{}
			}
			v.regions[region] = struct{}{}
		}
	}

	var item types.String
	diags := tfsdk.ValueAs(ctx, req.AttributeConfig, &item)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	if item.IsUnknown() || item.IsNull() {
		return
	}

	if _, ok := v.regions[item.ValueString()]; !ok {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Invalid Serverless Function Region",
			fmt.Sprintf("The serverless function region %s is not supported on Vercel. Must be one of %s.", item.ValueString(), strings.Join(keys(v.regions), ", ")),
		)
		return
	}
}
