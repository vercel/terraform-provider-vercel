package vercel

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestProviderCredentials(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config providerData
		env    map[string]string
		want   providerCredentials
	}{
		{
			name:   "configured API token over OIDC environment",
			config: providerData{APIToken: types.StringValue("config-token")},
			env:    map[string]string{oidcTokenEnv: "env-oidc-token"},
			want:   providerCredentials{apiToken: "config-token"},
		},
		{
			name:   "configured OIDC token over API token environment",
			config: providerData{OIDCToken: types.StringValue("config-oidc-token")},
			env:    map[string]string{"VERCEL_API_TOKEN": "env-token"},
			want:   providerCredentials{oidcToken: "config-oidc-token"},
		},
		{
			name:   "configured OIDC token file over environment",
			config: providerData{OIDCTokenFilePath: types.StringValue("/config/token")},
			env:    map[string]string{"VERCEL_API_TOKEN": "env-token", oidcTokenEnv: "env-oidc-token"},
			want:   providerCredentials{oidcTokenFilePath: "/config/token"},
		},
		{
			name: "API token environment over OIDC environment",
			env:  map[string]string{"VERCEL_API_TOKEN": "env-token", oidcTokenEnv: "env-oidc-token", oidcTokenFilePathEnv: "/env/token"},
			want: providerCredentials{apiToken: "env-token"},
		},
		{
			name: "OIDC token environment over OIDC token file environment",
			env:  map[string]string{oidcTokenEnv: "env-oidc-token", oidcTokenFilePathEnv: "/env/token"},
			want: providerCredentials{oidcToken: "env-oidc-token"},
		},
		{
			name: "OIDC token file environment",
			env:  map[string]string{oidcTokenFilePathEnv: "/env/token"},
			want: providerCredentials{oidcTokenFilePath: "/env/token"},
		},
		{name: "none"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, name := range []string{"VERCEL_API_TOKEN", oidcTokenEnv, oidcTokenFilePathEnv} {
				t.Setenv(name, tc.env[name])
			}
			if got := tc.config.credentials(); got != tc.want {
				t.Fatalf("credentials() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestProviderCredentialsReadOIDCTokenFileOnEveryCall(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	credentials := providerCredentials{oidcTokenFilePath: path}

	for _, token := range []string{"oidc-token-1", "oidc-token-2"} {
		if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := credentials.oidcSubjectToken()
		if err != nil || got != token {
			t.Fatalf("oidcSubjectToken() = %q, %v, want %q", got, err, token)
		}
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := credentials.oidcSubjectToken(); err == nil {
		t.Fatal("oidcSubjectToken() error = nil, want an error for a missing file")
	}
}

func TestProviderConfigWithoutNetworkAccess(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(New())()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	typ := schemas.Provider.ValueType()

	for _, tc := range []struct {
		name               string
		config             map[string]string
		wantValidateError  string
		wantConfigureError string
	}{
		{
			name:              "API token and OIDC token",
			config:            map[string]string{"api_token": "token", "oidc_token": "oidc-token"},
			wantValidateError: "Invalid Attribute Combination",
		},
		{
			name:              "OIDC token and OIDC token file",
			config:            map[string]string{"oidc_token": "oidc-token", "oidc_token_file_path": "/token"},
			wantValidateError: "Invalid Attribute Combination",
		},
		{
			name:               "OIDC token without team",
			config:             map[string]string{"oidc_token": "oidc-token", "oidc_policy_id": "pol_123"},
			wantConfigureError: "Unable to find team",
		},
		{
			name:               "OIDC token file without team",
			config:             map[string]string{"oidc_token_file_path": "/token"},
			wantConfigureError: "Unable to find team",
		},
		{
			name:               "no credentials",
			config:             map[string]string{"team": "acme"},
			wantConfigureError: "Unable to find api_token",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, name := range []string{"VERCEL_API_TOKEN", oidcTokenEnv, oidcTokenFilePathEnv, oidcPolicyIDEnv} {
				t.Setenv(name, "")
			}
			attributes := map[string]tftypes.Value{}
			for name, attributeType := range typ.(tftypes.Object).AttributeTypes {
				attributes[name] = tftypes.NewValue(attributeType, nil)
				if value, ok := tc.config[name]; ok {
					attributes[name] = tftypes.NewValue(attributeType, value)
				}
			}
			config, err := tfprotov6.NewDynamicValue(typ, tftypes.NewValue(typ, attributes))
			if err != nil {
				t.Fatal(err)
			}

			validated, err := server.ValidateProviderConfig(ctx, &tfprotov6.ValidateProviderConfigRequest{Config: &config})
			if err != nil {
				t.Fatal(err)
			}
			if got := errorSummary(validated.Diagnostics); got != tc.wantValidateError {
				t.Fatalf("validation error = %q, want %q: %v", got, tc.wantValidateError, validated.Diagnostics)
			}
			if tc.wantValidateError != "" {
				return
			}

			configured, err := server.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{Config: &config})
			if err != nil {
				t.Fatal(err)
			}
			if got := errorSummary(configured.Diagnostics); got != tc.wantConfigureError {
				t.Fatalf("configure error = %q, want %q: %v", got, tc.wantConfigureError, configured.Diagnostics)
			}
		})
	}
}

func errorSummary(diagnostics []*tfprotov6.Diagnostic) string {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			return diagnostic.Summary
		}
	}
	return ""
}
