package vercel

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

// TestConvertResponseToDNSRecord_InitializesSRVField ensures non-SRV records
// have their SRV field properly initialized to prevent "Value Conversion Error"
// Regression test for issue introduced in commit e7dd870
func TestConvertResponseToDNSRecord_InitializesSRVField(t *testing.T) {
	// Test a non-SRV record (CNAME)
	cnameRecord := client.DNSRecord{
		ID:         "test-id",
		Domain:     "example.com",
		Name:       "subdomain",
		RecordType: "CNAME",
		Value:      "target.example.com",
		TTL:        60,
	}

	result, err := convertResponseToDNSRecord(
		cnameRecord,
		types.StringValue("target.example.com"),
		types.ObjectNull(srvAttrType.AttrTypes),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The critical check: SRV must be initialized (not unknown) and must be null
	if result.SRV.IsUnknown() {
		t.Fatal("SRV field is unknown, should be null for non-SRV records")
	}
	if !result.SRV.IsNull() {
		t.Fatal("SRV field is not null for CNAME record")
	}
}
