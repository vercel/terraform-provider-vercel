package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var drainSamplingRuleAttrType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"rate":         types.Float64Type,
		"environment":  types.StringType,
		"request_path": types.StringType,
	},
}

type DrainSamplingRule struct {
	Rate        types.Float64 `tfsdk:"rate"`
	Environment types.String  `tfsdk:"environment"`
	RequestPath types.String  `tfsdk:"request_path"`
}

func drainSamplingRulesToClient(ctx context.Context, samplingRules types.List) ([]client.DrainSamplingRule, diag.Diagnostics) {
	var result []client.DrainSamplingRule
	if samplingRules.IsNull() || samplingRules.IsUnknown() {
		return result, nil
	}

	var rules []DrainSamplingRule
	diags := samplingRules.ElementsAs(ctx, &rules, false)
	if diags.HasError() {
		return result, diags
	}

	result = make([]client.DrainSamplingRule, 0, len(rules))
	for _, rule := range rules {
		result = append(result, client.DrainSamplingRule{
			Rate:        rule.Rate.ValueFloat64(),
			Environment: rule.Environment.ValueString(),
			RequestPath: rule.RequestPath.ValueString(),
		})
	}
	return result, nil
}

func drainSamplingRulesFromAPI(ctx context.Context, apiRules []client.DrainSamplingRule, preferredList types.List) (types.List, diag.Diagnostics) {
	if len(apiRules) == 0 {
		if preferredList.IsNull() || preferredList.IsUnknown() {
			return types.ListNull(drainSamplingRuleAttrType), nil
		}
		return types.ListValueMust(drainSamplingRuleAttrType, []attr.Value{}), nil
	}

	rules := make([]DrainSamplingRule, 0, len(apiRules))
	for _, rule := range apiRules {
		environment := types.StringNull()
		if rule.Environment != "" {
			environment = types.StringValue(rule.Environment)
		}
		requestPath := types.StringNull()
		if rule.RequestPath != "" {
			requestPath = types.StringValue(rule.RequestPath)
		}
		rules = append(rules, DrainSamplingRule{
			Rate:        types.Float64Value(rule.Rate),
			Environment: environment,
			RequestPath: requestPath,
		})
	}

	return types.ListValueFrom(ctx, drainSamplingRuleAttrType, rules)
}
