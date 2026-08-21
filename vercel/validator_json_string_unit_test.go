package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidatorJSONObject(t *testing.T) {
	for _, tc := range []struct {
		name      string
		value     types.String
		wantError bool
	}{
		{name: "null is allowed", value: types.StringNull(), wantError: false},
		{name: "unknown is allowed", value: types.StringUnknown(), wantError: false},
		{name: "object is allowed", value: types.StringValue(`{"aud":"https://example.com"}`), wantError: false},
		{name: "empty object is allowed", value: types.StringValue(`{}`), wantError: false},
		{name: "empty string is rejected", value: types.StringValue(""), wantError: true},
		{name: "invalid json is rejected", value: types.StringValue("{not json"), wantError: true},
		{name: "json array is rejected", value: types.StringValue(`["a","b"]`), wantError: true},
		{name: "json string scalar is rejected", value: types.StringValue(`"a string"`), wantError: true},
		{name: "json null literal is rejected", value: types.StringValue("null"), wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("token_claims"),
				ConfigValue: tc.value,
			}
			resp := &validator.StringResponse{}
			validateJSONObject().ValidateString(context.Background(), req, resp)

			if tc.wantError && !resp.Diagnostics.HasError() {
				t.Fatalf("expected a validation error, got none")
			}
			if !tc.wantError && resp.Diagnostics.HasError() {
				t.Fatalf("expected no validation error, got: %s", resp.Diagnostics)
			}
		})
	}
}
