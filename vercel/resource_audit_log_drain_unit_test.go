package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestAuditLogDrainDeliverySchemaUsesOptionalNestedAttributes(t *testing.T) {
	var response resource.SchemaResponse
	newAuditLogDrainResource().Schema(context.Background(), resource.SchemaRequest{}, &response)

	requiredAttributes := map[string][]string{
		"http": {"endpoint", "encoding"},
		"s3":   {"endpoint", "encoding", "role_arn", "region"},
	}
	for name, required := range requiredAttributes {
		attribute, ok := response.Schema.Attributes[name].(schema.SingleNestedAttribute)
		if !ok {
			t.Fatalf("%s schema = %T, want schema.SingleNestedAttribute", name, response.Schema.Attributes[name])
		}
		if !attribute.IsOptional() {
			t.Fatalf("%s schema must be optional", name)
		}
		for _, childName := range required {
			if !attribute.Attributes[childName].IsRequired() {
				t.Fatalf("%s.%s schema must be required", name, childName)
			}
		}
	}
}

func TestAuditLogDrainFromAPIPreservesOmittedSensitiveHTTPValues(t *testing.T) {
	prior := AuditLogDrain{
		HTTP: &AuditLogDrainHTTPConfig{
			Headers: types.MapValueMust(types.StringType, map[string]attr.Value{
				"Authorization": types.StringValue("Bearer token"),
			}),
			Secret:      types.StringValue("a_very_long_and_very_well_specified_secret"),
			Compression: types.StringValue("gzip"),
		},
	}
	out := client.AuditLogDrain{
		ID:     "drn_123",
		TeamID: "team_123",
		Name:   "security",
		HTTP: &client.AuditLogDrainHTTPDelivery{
			Endpoint: "https://example.com/audit",
			Encoding: "json",
		},
	}

	result, err := auditLogDrainFromAPI(context.Background(), out, &prior)
	if err != nil {
		t.Fatalf("auditLogDrainFromAPI() error = %v", err)
	}
	if result.HTTP.Secret.ValueString() != prior.HTTP.Secret.ValueString() {
		t.Fatalf("secret = %q, want prior value", result.HTTP.Secret.ValueString())
	}
	if !result.HTTP.Headers.Equal(prior.HTTP.Headers) {
		t.Fatalf("headers = %#v, want %#v", result.HTTP.Headers, prior.HTTP.Headers)
	}
	if result.HTTP.Compression.ValueString() != "gzip" {
		t.Fatalf("compression = %q, want prior value", result.HTTP.Compression.ValueString())
	}
}

func TestAuditLogDrainFromAPINormalizesEmptyImportedHeaders(t *testing.T) {
	out := client.AuditLogDrain{
		ID:     "drn_123",
		TeamID: "team_123",
		Name:   "security",
		HTTP: &client.AuditLogDrainHTTPDelivery{
			Endpoint: "https://example.com/audit",
			Encoding: "json",
			Headers:  map[string]string{},
		},
	}

	result, err := auditLogDrainFromAPI(context.Background(), out, nil)
	if err != nil {
		t.Fatalf("auditLogDrainFromAPI() error = %v", err)
	}
	if !result.HTTP.Headers.IsNull() {
		t.Fatalf("headers = %#v, want null", result.HTTP.Headers)
	}
}
