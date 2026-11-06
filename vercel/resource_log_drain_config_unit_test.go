package vercel_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/vercel/terraform-provider-vercel/v5/vercel"
)

func TestLogDrainSecretConfiguration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := providerserver.NewProtocol6(vercel.New())()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	const resourceName = "vercel_log_drain"
	typ := schemas.ResourceSchemas[resourceName].ValueType()
	rootType := typ.(tftypes.Object)

	for _, tc := range []struct {
		name       string
		secret     any
		secretWO   any
		version    any
		wantErrors bool
	}{
		{name: "generated secret"},
		{name: "legacy secret", secret: "legacy-secret-that-is-at-least-thirty-two-characters"},
		{name: "write-only secret", secretWO: "write-only-secret-that-is-at-least-thirty-two-characters", version: int64(1)},
		{name: "secret conflict", secret: "legacy-secret-that-is-at-least-thirty-two-characters", secretWO: "write-only-secret-that-is-at-least-thirty-two-characters", version: int64(1), wantErrors: true},
		{name: "zero version", secretWO: "write-only-secret-that-is-at-least-thirty-two-characters", version: int64(0)},
		{name: "negative version", secretWO: "write-only-secret-that-is-at-least-thirty-two-characters", version: int64(-1), wantErrors: true},
		{name: "write-only secret without version", secretWO: "write-only-secret-that-is-at-least-thirty-two-characters"},
		{name: "version without write-only secret", version: int64(1), wantErrors: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := make(map[string]tftypes.Value, len(rootType.AttributeTypes))
			for name, typ := range rootType.AttributeTypes {
				config[name] = tftypes.NewValue(typ, nil)
			}
			config["name"] = tftypes.NewValue(tftypes.String, "write-only-secret-test")
			config["delivery_format"] = tftypes.NewValue(tftypes.String, "ndjson")
			config["environments"] = logDrainStringSet(rootType.AttributeTypes["environments"], "production")
			config["sources"] = logDrainStringSet(rootType.AttributeTypes["sources"], "static")
			config["endpoint"] = tftypes.NewValue(tftypes.String, "https://example.com/logs")
			config["secret"] = tftypes.NewValue(tftypes.String, tc.secret)
			config["secret_wo"] = tftypes.NewValue(tftypes.String, tc.secretWO)
			config["secret_wo_version"] = tftypes.NewValue(tftypes.Number, tc.version)
			configValue, err := tfprotov6.NewDynamicValue(typ, tftypes.NewValue(typ, config))
			if err != nil {
				t.Fatal(err)
			}

			validation, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
				TypeName: resourceName,
				Config:   &configValue,
				ClientCapabilities: &tfprotov6.ValidateResourceConfigClientCapabilities{
					WriteOnlyAttributesAllowed: true,
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			gotErrors := hasLogDrainError(validation.Diagnostics)
			if gotErrors != tc.wantErrors {
				for _, diag := range validation.Diagnostics {
					t.Logf("%s: %s", diag.Summary, diag.Detail)
				}
				t.Fatalf("validation errors = %t, want %t", gotErrors, tc.wantErrors)
			}
		})
	}
}

func logDrainStringSet(typ tftypes.Type, values ...string) tftypes.Value {
	items := make([]tftypes.Value, len(values))
	for index, value := range values {
		items[index] = tftypes.NewValue(tftypes.String, value)
	}
	return tftypes.NewValue(typ, items)
}

func hasLogDrainError(diags []*tfprotov6.Diagnostic) bool {
	for _, diag := range diags {
		if diag.Severity == tfprotov6.DiagnosticSeverityError {
			return true
		}
	}
	return false
}

func TestLogDrainSamplingConfiguration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := providerserver.NewProtocol6(vercel.New())()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	const resourceName = "vercel_log_drain"
	typ := schemas.ResourceSchemas[resourceName].ValueType()
	rootType := typ.(tftypes.Object)
	rulesType := rootType.AttributeTypes["sampling_rules"].(tftypes.List)
	type rule struct {
		rate        any
		environment any
		requestPath any
	}
	tooManyRules := make([]rule, 11)
	for i := range tooManyRules {
		tooManyRules[i] = rule{rate: 0.5}
	}

	for _, tc := range []struct {
		name       string
		rate       any
		rules      []rule
		wantErrors bool
	}{
		{name: "no sampling"},
		{name: "legacy rate", rate: 0.5},
		{name: "legacy zero", rate: 0.0},
		{name: "empty rules", rules: []rule{}},
		{
			name: "ordered rules with zero and fallback",
			rules: []rule{
				{rate: 0.0, environment: "production", requestPath: "/health"},
				{rate: 0.25, environment: "production"},
				{rate: 1.0},
			},
		},
		{name: "preview rule", rules: []rule{{rate: 0.5, environment: "preview"}}},
		{name: "rate conflicts with rules", rate: 0.5, rules: []rule{{rate: 0.25}}, wantErrors: true},
		{name: "rate conflicts with empty rules", rate: 0.0, rules: []rule{}, wantErrors: true},
		{name: "negative rule rate", rules: []rule{{rate: -0.1}}, wantErrors: true},
		{name: "rule rate exceeds one", rules: []rule{{rate: 1.1}}, wantErrors: true},
		{name: "missing rule rate", rules: []rule{{environment: "production"}}, wantErrors: true},
		{name: "empty request path", rules: []rule{{rate: 0.5, requestPath: ""}}, wantErrors: true},
		{name: "invalid environment", rules: []rule{{rate: 0.5, environment: "development"}}, wantErrors: true},
		{name: "more than ten rules", rules: tooManyRules, wantErrors: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := make(map[string]tftypes.Value, len(rootType.AttributeTypes))
			for name, attributeType := range rootType.AttributeTypes {
				config[name] = tftypes.NewValue(attributeType, nil)
			}
			config["name"] = tftypes.NewValue(tftypes.String, "sampling-test")
			config["delivery_format"] = tftypes.NewValue(tftypes.String, "ndjson")
			config["environments"] = logDrainStringSet(rootType.AttributeTypes["environments"], "production", "preview")
			config["sources"] = logDrainStringSet(rootType.AttributeTypes["sources"], "lambda")
			config["endpoint"] = tftypes.NewValue(tftypes.String, "https://example.com/logs")
			config["sampling_rate"] = tftypes.NewValue(tftypes.Number, tc.rate)
			if tc.rules != nil {
				values := make([]tftypes.Value, len(tc.rules))
				for i, rule := range tc.rules {
					values[i] = tftypes.NewValue(rulesType.ElementType, map[string]tftypes.Value{
						"rate":         tftypes.NewValue(tftypes.Number, rule.rate),
						"environment":  tftypes.NewValue(tftypes.String, rule.environment),
						"request_path": tftypes.NewValue(tftypes.String, rule.requestPath),
					})
				}
				config["sampling_rules"] = tftypes.NewValue(rulesType, values)
			}
			configValue, err := tfprotov6.NewDynamicValue(typ, tftypes.NewValue(typ, config))
			if err != nil {
				t.Fatal(err)
			}
			validation, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
				TypeName: resourceName,
				Config:   &configValue,
				ClientCapabilities: &tfprotov6.ValidateResourceConfigClientCapabilities{
					WriteOnlyAttributesAllowed: true,
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if gotErrors := hasLogDrainError(validation.Diagnostics); gotErrors != tc.wantErrors {
				for _, diag := range validation.Diagnostics {
					t.Logf("%s: %s", diag.Summary, diag.Detail)
				}
				t.Fatalf("validation errors = %t, want %t", gotErrors, tc.wantErrors)
			}
		})
	}
}
