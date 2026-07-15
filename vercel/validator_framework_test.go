package vercel

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateFrameworkAllowsServices(t *testing.T) {
	frameworkValidator := validateFramework()
	if frameworkValidator.frameworksURL != "https://api-frameworks.vercel.sh/api/v1/frameworks?includeExperimental=true" {
		t.Fatalf("expected framework validation to include experimental frameworks, got %s", frameworkValidator.frameworksURL)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[{"slug":"services"}]`)
	}))
	t.Cleanup(server.Close)
	frameworkValidator.frameworksURL = server.URL

	var resp validator.StringResponse
	frameworkValidator.ValidateString(context.Background(), validator.StringRequest{
		ConfigValue: types.StringValue("services"),
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected services framework to be valid, got diagnostics: %v", resp.Diagnostics)
	}
}
