package vercel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const testFrameworkListResponse = `[{"slug":"nextjs"},{"slug":"vite"},{"slug":null}]`

func TestValidatorFramework(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(testFrameworkListResponse))
	}))
	t.Cleanup(server.Close)

	for _, tc := range []struct {
		name      string
		value     types.String
		wantError bool
	}{
		{name: "null is allowed", value: types.StringNull(), wantError: false},
		{name: "unknown is allowed", value: types.StringUnknown(), wantError: false},
		{name: "framework from the api is allowed", value: types.StringValue("nextjs"), wantError: false},
		{name: "ash is allowed even though the api omits it", value: types.StringValue("ash"), wantError: false},
		{name: "unsupported framework is rejected", value: types.StringValue("notexist"), wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			validatorFramework{url: server.URL}.ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("framework"),
				ConfigValue: tc.value,
			}, resp)

			if tc.wantError && !resp.Diagnostics.HasError() {
				t.Fatalf("expected a validation error, got none")
			}
			if !tc.wantError && resp.Diagnostics.HasError() {
				t.Fatalf("expected no validation error, got: %s", resp.Diagnostics)
			}
		})
	}
}

func TestValidatorFrameworkDefaultsToVercelAPI(t *testing.T) {
	if validateFramework().url != frameworksAPIURL {
		t.Fatalf("expected the validator to default to %s, got %s", frameworksAPIURL, validateFramework().url)
	}
}

func TestValidatorFrameworkNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	resp := &validator.StringResponse{}
	validatorFramework{url: server.URL}.ValidateString(context.Background(), validator.StringRequest{
		Path:        path.Root("framework"),
		ConfigValue: types.StringValue("ash"),
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a validation error for a non-200 response, got none")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Unable to validate attribute" {
		t.Fatalf("unexpected error summary: %s", got)
	}
}

func TestValidatorFrameworkNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	resp := &validator.StringResponse{}
	validatorFramework{url: url}.ValidateString(context.Background(), validator.StringRequest{
		Path:        path.Root("framework"),
		ConfigValue: types.StringValue("ash"),
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a validation error when the frameworks endpoint is unreachable, got none")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Unable to validate attribute" {
		t.Fatalf("unexpected error summary: %s", got)
	}
}
