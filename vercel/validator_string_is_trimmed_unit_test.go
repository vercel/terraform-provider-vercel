package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidatorStringIsTrimmed(t *testing.T) {
	for _, test := range []struct {
		name      string
		value     types.String
		wantError bool
	}{
		{name: "trimmed", value: types.StringValue("value")},
		{name: "leading whitespace", value: types.StringValue(" value"), wantError: true},
		{name: "trailing whitespace", value: types.StringValue("value "), wantError: true},
		{name: "unknown", value: types.StringUnknown()},
		{name: "null", value: types.StringNull()},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := &validator.StringResponse{}
			validateStringIsTrimmed().ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: test.value,
				Path:        path.Root("value"),
			}, response)
			if response.Diagnostics.HasError() != test.wantError {
				t.Fatalf("HasError() = %t, want %t; diagnostics = %v", response.Diagnostics.HasError(), test.wantError, response.Diagnostics)
			}
		})
	}
}
