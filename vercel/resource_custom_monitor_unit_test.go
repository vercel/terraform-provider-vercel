package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestCustomMonitorSchemaMirrorsCustomAlertRule(t *testing.T) {
	var response resource.SchemaResponse
	newCustomMonitorResource().Schema(context.Background(), resource.SchemaRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	monitor := response.Schema
	rule := customAlertRuleSchema(t)

	if len(monitor.Attributes) != len(rule.Attributes) {
		t.Fatalf("custom monitor has %d attributes, custom alert rule has %d", len(monitor.Attributes), len(rule.Attributes))
	}
	for name, attribute := range rule.Attributes {
		got, ok := monitor.Attributes[name]
		if !ok {
			t.Fatalf("custom monitor schema is missing %q", name)
		}
		if got.IsRequired() != attribute.IsRequired() || got.IsOptional() != attribute.IsOptional() || got.IsComputed() != attribute.IsComputed() {
			t.Errorf("%s differs from the custom alert rule attribute", name)
		}
	}
	for _, name := range []string{"id", "name", "tags", "notification_settings"} {
		if got := monitor.Attributes[name].GetMarkdownDescription(); strings.Contains(got, "alert rule") || strings.Contains(got, "the rule") {
			t.Errorf("%s description %q still uses alert rule wording", name, got)
		}
	}
	if monitor.DeprecationMessage != "" {
		t.Errorf("custom monitor DeprecationMessage = %q, want empty", monitor.DeprecationMessage)
	}
	if !strings.Contains(rule.DeprecationMessage, "vercel_custom_monitor") {
		t.Errorf("custom alert rule DeprecationMessage = %q, want it to point to vercel_custom_monitor", rule.DeprecationMessage)
	}
}

func TestCustomMonitorMetadata(t *testing.T) {
	var response resource.MetadataResponse
	newCustomMonitorResource().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "vercel"}, &response)
	if response.TypeName != "vercel_custom_monitor" {
		t.Fatalf("TypeName = %q, want vercel_custom_monitor", response.TypeName)
	}
}

func TestCustomMonitorCreateRequiresTeam(t *testing.T) {
	ctx := context.Background()
	model := customAlertTestModel(t)
	model.TeamID = types.StringNull()
	var schemaResponse resource.SchemaResponse
	newCustomMonitorResource().Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	plan := tfsdk.Plan{Schema: schemaResponse.Schema}
	if diags := plan.Set(ctx, model); diags.HasError() {
		t.Fatal(diags)
	}
	var response resource.CreateResponse
	(&customMonitorResource{client: client.New("TOKEN")}).Create(ctx, resource.CreateRequest{Plan: plan}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("creation without a team was accepted")
	}
}

// The lifecycle must use the monitors API and its `monitor` envelope.
func TestCustomMonitorLifecycle(t *testing.T) {
	var mu sync.Mutex
	var monitor client.AlertRule
	var tags json.RawMessage
	var deleted bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if req.URL.Query().Get("teamId") != "team_123" {
			t.Errorf("teamId = %q", req.URL.Query().Get("teamId"))
		}
		switch req.Method {
		case http.MethodPost:
			if req.URL.Path != "/alerts/v3/monitors" {
				t.Errorf("create path = %s", req.URL.Path)
			}
			if err := json.NewDecoder(req.Body).Decode(&monitor); err != nil {
				t.Error(err)
				http.Error(w, "invalid", 400)
				return
			}
			tags, _ = json.Marshal(monitor.Tags)
			monitor.ID, monitor.QuerySupported = "ar_123", true
		case http.MethodPatch:
			if req.URL.Path != "/alerts/v3/monitors/ar_123" {
				t.Errorf("update path = %s", req.URL.Path)
			}
			var patch struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(req.Body).Decode(&patch); err != nil {
				t.Error(err)
			}
			monitor.Name = patch.Name
		case http.MethodGet:
			if req.URL.Path != "/alerts/v3/monitors/ar_123" {
				t.Errorf("read path = %s", req.URL.Path)
			}
			if deleted {
				http.Error(w, `{"error":{"code":"not_found","message":"missing"}}`, 404)
				return
			}
		case http.MethodDelete:
			if req.URL.Path != "/alerts/v3/monitors/ar_123" {
				t.Errorf("delete path = %s", req.URL.Path)
			}
			deleted = true
			_, _ = fmt.Fprint(w, `{"success":true}`)
			return
		default:
			t.Errorf("unexpected method %s", req.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"monitor": monitor})
	}))
	t.Cleanup(server.Close)
	factories := map[string]func() (tfprotov6.ProviderServer, error){"vercel": providerserver.NewProtocol6WithError(&customAlertMockProvider{
		Provider: New(), client: client.New("TOKEN").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_123"}),
	})}
	config := func(name string) string {
		return fmt.Sprintf(`
resource "vercel_custom_monitor" "test" {
 name = %q
 project_id = "prj_123"
 severity = "high"
 evaluation = {
  window = "5m"
  query = {metrics = {requests = {metric = "vercel.request.count", aggregation = "count"}}, outputs = ["requests"]}
 }
 trigger = {type = "threshold", output = "requests", operator = "gt", threshold = 0}
}
`, name)
	}
	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: factories, Steps: []tfresource.TestStep{
		{Config: config("Requests"), Check: tfresource.ComposeTestCheckFunc(
			tfresource.TestCheckResourceAttr("vercel_custom_monitor.test", "id", "ar_123"),
			tfresource.TestCheckResourceAttr("vercel_custom_monitor.test", "team_id", "team_123"),
		)},
		{Config: config("Renamed")},
		{ResourceName: "vercel_custom_monitor.test", ImportState: true, ImportStateId: "team_123/ar_123", ImportStateVerify: true, ImportStateVerifyIgnore: []string{"evaluation.query"}},
	}})
	mu.Lock()
	defer mu.Unlock()
	if !deleted {
		t.Fatal("monitor was not deleted")
	}
	if string(tags) != `["terraform-provider-vercel"]` {
		t.Fatalf("create tags = %s, want the provider-owned tag", tags)
	}
}
