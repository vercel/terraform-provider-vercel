package vercel

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestFirewallConfigDeleteLeavesFirewallEnabledAndEmpty(t *testing.T) {
	var (
		method string
		path   string
		body   map[string]any
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("invalid request body %q: %s", raw, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"active":{"firewallEnabled":true}}`))
	}))
	t.Cleanup(server.Close)

	ctx := context.Background()
	r := &firewallConfigResource{client: client.New("INVALID").WithBaseURL(server.URL)}

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	state := tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
	}
	diags := state.Set(ctx, &FirewallConfig{
		ID:        types.StringValue("team_123/prj_123"),
		ProjectID: types.StringValue("prj_123"),
		TeamID:    types.StringValue("team_123"),
		Enabled:   types.BoolValue(true),
	})
	if diags.HasError() {
		t.Fatalf("setting state: %v", diags)
	}

	resp := &resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete() diagnostics: %v", resp.Diagnostics)
	}

	if method != http.MethodPut || path != "/v1/security/firewall/config" {
		t.Fatalf("request = %s %s, want PUT /v1/security/firewall/config", method, path)
	}
	if enabled, ok := body["firewallEnabled"].(bool); !ok || !enabled {
		t.Fatalf("firewallEnabled = %v, want true", body["firewallEnabled"])
	}
	for _, key := range []string{"rules", "ips", "managedRules", "crs"} {
		if _, ok := body[key]; ok {
			t.Errorf("body contains %q, want it omitted so the config is empty", key)
		}
	}
}
