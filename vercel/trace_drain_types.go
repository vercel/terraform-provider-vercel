package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

var traceDrainSamplingRuleAttrType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"rate":         types.Float64Type,
		"environment":  types.StringType,
		"request_path": types.StringType,
	},
}

type TraceDrainSamplingRule struct {
	Rate        types.Float64 `tfsdk:"rate"`
	Environment types.String  `tfsdk:"environment"`
	RequestPath types.String  `tfsdk:"request_path"`
}

func traceDrainSamplingRulesToClient(ctx context.Context, samplingRules types.List) ([]client.TraceDrainSamplingRule, diag.Diagnostics) {
	var result []client.TraceDrainSamplingRule
	if samplingRules.IsNull() || samplingRules.IsUnknown() {
		return result, nil
	}

	var rules []TraceDrainSamplingRule
	diags := samplingRules.ElementsAs(ctx, &rules, false)
	if diags.HasError() {
		return result, diags
	}

	result = make([]client.TraceDrainSamplingRule, 0, len(rules))
	for _, rule := range rules {
		result = append(result, client.TraceDrainSamplingRule{
			Rate:        rule.Rate.ValueFloat64(),
			Environment: rule.Environment.ValueString(),
			RequestPath: rule.RequestPath.ValueString(),
		})
	}
	return result, nil
}

func traceDrainSamplingRulesFromAPI(ctx context.Context, apiRules []client.TraceDrainSamplingRule, preferredList types.List) (types.List, diag.Diagnostics) {
	if len(apiRules) == 0 {
		if preferredList.IsNull() || preferredList.IsUnknown() {
			return types.ListNull(traceDrainSamplingRuleAttrType), nil
		}
		return types.ListValueMust(traceDrainSamplingRuleAttrType, []attr.Value{}), nil
	}

	rules := make([]TraceDrainSamplingRule, 0, len(apiRules))
	for _, rule := range apiRules {
		environment := types.StringNull()
		if rule.Environment != "" {
			environment = types.StringValue(rule.Environment)
		}
		requestPath := types.StringNull()
		if rule.RequestPath != "" {
			requestPath = types.StringValue(rule.RequestPath)
		}
		rules = append(rules, TraceDrainSamplingRule{
			Rate:        types.Float64Value(rule.Rate),
			Environment: environment,
			RequestPath: requestPath,
		})
	}

	return types.ListValueFrom(ctx, traceDrainSamplingRuleAttrType, rules)
}
