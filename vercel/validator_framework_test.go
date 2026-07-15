package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateFrameworkAllowsServices(t *testing.T) {
	var resp validator.StringResponse

	validateFramework().ValidateString(context.Background(), validator.StringRequest{
		ConfigValue: types.StringValue(servicesFramework),
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected services framework to be valid, got diagnostics: %v", resp.Diagnostics)
	}
}
