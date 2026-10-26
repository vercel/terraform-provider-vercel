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
		{name: "write-only secret without version", secretWO: "write-only-secret-that-is-at-least-thirty-two-characters"},
		{name: "version without write-only secret", version: int64(1), wantErrors: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := developmentSecretObject(rootType)
			config["name"] = tftypes.NewValue(tftypes.String, "write-only-secret-test")
			config["delivery_format"] = tftypes.NewValue(tftypes.String, "ndjson")
			config["environments"] = logDrainStringSet(rootType.AttributeTypes["environments"], "production")
			config["sources"] = logDrainStringSet(rootType.AttributeTypes["sources"], "static")
			config["endpoint"] = tftypes.NewValue(tftypes.String, "https://example.com/logs")
			config["secret"] = tftypes.NewValue(tftypes.String, tc.secret)
			config["secret_wo"] = tftypes.NewValue(tftypes.String, tc.secretWO)
			config["secret_wo_version"] = tftypes.NewValue(tftypes.Number, tc.version)
			configValue := developmentSecretDynamicValue(t, typ, tftypes.NewValue(typ, config))

			validation, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
				TypeName: resourceName,
				Config:   configValue,
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
