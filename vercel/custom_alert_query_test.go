package vercel

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestCustomAlertQueryJSONRoundTrip(t *testing.T) {
	for _, query := range []string{
		`{"metrics":{"requests":{"metric":"vercel.request.count","aggregation":"count"}},"outputs":["requests"]}`,
		`{"metrics":{"requests":{"metric":"vercel.request.count","aggregation":"count","per":"second","filter":"httpStatus >= 500"}},"outputs":["requests"],"groupBy":["route"],"filter":"environment:production"}`,
		`{"metrics":{"errors":{"metric":"vercel.request.count","aggregation":"count","normalize":"percent","filter":"httpStatus >= 500"},"requests":{"metric":"vercel.request.count","aggregation":"count"}},"formulas":{"formula":"errors / requests"},"outputs":["formula"]}`,
		`{"metrics":{"visitors":{"metric":"vercel.request.count","aggregation":"unique","dimensions":["clientIp"]}},"outputs":["visitors"]}`,
	} {
		t.Run(query, func(t *testing.T) {
			ctx := context.Background()
			value, diags := customAlertQueryFromJSON(ctx, json.RawMessage(query))
			if diags.HasError() {
				t.Fatal(diags)
			}
			encoded, diags := customAlertQueryToJSON(ctx, value)
			if diags.HasError() {
				t.Fatal(diags)
			}
			before, _ := normalizeJSON(query)
			after, err := normalizeJSON(string(encoded))
			if err != nil || before != after {
				t.Fatalf("query round trip = %s, want %s: %v", encoded, query, err)
			}
		})
	}
}

func TestCustomAlertQuerySchemaValidation(t *testing.T) {
	const metric = `{"metric":"vercel.request.count","aggregation":"count"}`
	primitive := `{"metrics":{"requests":` + metric + `},"outputs":["requests"]}`
	for _, test := range []struct{ name, query, errorPath string }{
		{"primitive", primitive, ""},
		{"ratio", `{"metrics":{"errors":` + metric + `,"requests":` + metric + `},"formulas":{"formula":"errors / requests"},"outputs":["formula"]}`, ""},
		{"invalid aggregation", strings.Replace(primitive, `"count"`, `"invalid"`, 1), "aggregation"},
		{"invalid rate", strings.Replace(primitive, `"aggregation":"count"`, `"aggregation":"count","per":"minute"`, 1), "per"},
		{"too many grouping dimensions", strings.Replace(primitive, `"outputs"`, `"groupBy":["route","region"],"outputs"`, 1), "group_by"},
		{"wrong primitive output", strings.Replace(primitive, `["requests"]`, `["missing"]`, 1), "outputs"},
		{"two metrics without formula", `{"metrics":{"errors":` + metric + `,"requests":` + metric + `},"outputs":["errors"]}`, "query"},
		{"formula references missing metric", `{"metrics":{"errors":` + metric + `,"requests":` + metric + `},"formulas":{"formula":"errors / missing"},"outputs":["formula"]}`, "query"},
		{"reserved metric alias", strings.ReplaceAll(primitive, "requests", "formula"), "metrics"},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := customAlertTestModel()
			model.ID = types.StringNull()
			model.Evaluation = customAlertEvaluationValue("5m", test.query)
			config := tfsdk.Plan{Schema: customAlertRuleSchema(t)}
			if diags := config.Set(context.Background(), model); diags.HasError() {
				t.Fatal(diags)
			}
			dynamic, err := tfprotov6.NewDynamicValue(config.Raw.Type(), config.Raw)
			if err != nil {
				t.Fatal(err)
			}
			response, err := providerserver.NewProtocol6(New())().ValidateResourceConfig(context.Background(), &tfprotov6.ValidateResourceConfigRequest{TypeName: "vercel_custom_alert_rule", Config: &dynamic})
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, d := range response.Diagnostics {
				if d.Severity != tfprotov6.DiagnosticSeverityError {
					continue
				}
				if test.errorPath == "" {
					t.Fatalf("unexpected diagnostic: %s: %s", d.Summary, d.Detail)
				}
				if d.Attribute != nil && strings.Contains(d.Attribute.String(), test.errorPath) {
					found = true
				}
			}
			if test.errorPath != "" && !found {
				t.Fatalf("expected an error at %s, got %v", test.errorPath, response.Diagnostics)
			}
		})
	}
}

func TestCustomAlertQueryUnknownMetric(t *testing.T) {
	ctx := context.Background()
	model := customAlertTestModel()
	model.ID = types.StringNull()
	evaluation := model.Evaluation.Attributes()
	query := evaluation["query"].(types.Object).Attributes()
	metrics := query["metrics"].(types.Map).Elements()
	metric := metrics["requests"].(types.Object).Attributes()
	metric["metric"] = types.StringUnknown()
	metrics["requests"] = types.ObjectValueMust(customAlertQueryMetricAttrTypes, metric)
	query["metrics"] = types.MapValueMust(types.ObjectType{AttrTypes: customAlertQueryMetricAttrTypes}, metrics)
	unknown := types.ObjectValueMust(customAlertQueryAttrTypes, query)
	evaluation["query"] = unknown
	model.Evaluation = types.ObjectValueMust(customAlertEvaluationAttrTypes, evaluation)
	config := tfsdk.Plan{Schema: customAlertRuleSchema(t)}
	if diags := config.Set(ctx, model); diags.HasError() {
		t.Fatal(diags)
	}
	dynamic, err := tfprotov6.NewDynamicValue(config.Raw.Type(), config.Raw)
	if err != nil {
		t.Fatal(err)
	}
	response, err := providerserver.NewProtocol6(New())().ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{TypeName: "vercel_custom_alert_rule", Config: &dynamic})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range response.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("unknown metric rejected during validation: %s", d.Detail)
		}
	}
	if _, diags := customAlertQueryToJSON(ctx, unknown); !diags.HasError() {
		t.Fatal("unresolved metric must not be serialized during apply")
	}
	// An entirely unknown query must also remain valid until its dependencies resolve.
	evaluation["query"] = types.ObjectUnknown(customAlertQueryAttrTypes)
	model.Evaluation = types.ObjectValueMust(customAlertEvaluationAttrTypes, evaluation)
	if diags := config.Set(ctx, model); diags.HasError() {
		t.Fatal(diags)
	}
	dynamic, err = tfprotov6.NewDynamicValue(config.Raw.Type(), config.Raw)
	if err != nil {
		t.Fatal(err)
	}
	response, err = providerserver.NewProtocol6(New())().ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{TypeName: "vercel_custom_alert_rule", Config: &dynamic})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range response.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("unknown query rejected during validation: %s", d.Detail)
		}
	}
}
