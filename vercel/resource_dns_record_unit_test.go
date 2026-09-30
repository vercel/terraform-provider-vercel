package vercel

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestHTTPSRecordResponsePreservesParams(t *testing.T) {
	for _, tc := range []struct {
		name   string
		value  string
		params types.String
	}{
		{"omitted", "1 example.com.", types.StringNull()},
		{"empty", "1 example.com. ", types.StringValue("")},
		{"spaces", "1 example.com.  alpn=h2  port=8443 ", types.StringValue(" alpn=h2  port=8443 ")},
		{"only spaces", "1 example.com.   ", types.StringValue("  ")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			planned := types.ObjectValueMust(httpsAttrType.AttrTypes, map[string]attr.Value{
				"priority": types.Int64Value(1),
				"target":   types.StringValue("Example.COM"),
				"params":   tc.params,
			})
			got, err := convertResponseToDNSRecord(client.DNSRecord{RecordType: "HTTPS", Value: tc.value}, types.StringNull(), types.ObjectNull(srvAttrType.AttrTypes), planned)
			if err != nil {
				t.Fatal(err)
			}
			if !got.HTTPS.Equal(planned) {
				t.Fatalf("HTTPS = %s, want %s", got.HTTPS, planned)
			}
			if !got.Value.IsNull() || !got.SRV.IsNull() {
				t.Fatal("HTTPS records must leave value and srv null")
			}
		})
	}
}
