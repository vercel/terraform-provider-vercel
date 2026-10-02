package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

type shareableTestProvider struct{ client *client.Client }

func (p *shareableTestProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "vercel"
}
func (p *shareableTestProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{}
}
func (p *shareableTestProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	resp.ResourceData = p.client
	resp.DataSourceData = p.client
}
func (p *shareableTestProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{newShareableLinkResource}
}
func (p *shareableTestProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{newShareableLinkDataSource}
}

func TestAcc_ShareableLinkTerraformLifecycle(t *testing.T) {
	var mu sync.Mutex
	bypasses := map[string]client.ProtectionBypass{"*": {Scope: "alias-protection-override"}}
	sequence := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if req.Method == "GET" {
			json.NewEncoder(w).Encode(map[string]any{"uid": "alias_123", "alias": "preview.vercel.app", "projectId": "prj_123", "protectionBypass": bypasses})
			return
		}
		if req.Method != "PATCH" || req.URL.Path != "/aliases/alias_123/protection-bypass" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			w.WriteHeader(400)
			return
		}
		var body struct {
			TTL    *int64 `json:"ttl"`
			Revoke *struct {
				Secret     string `json:"secret"`
				Regenerate bool   `json:"regenerate"`
			} `json:"revoke"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body.Revoke != nil {
			delete(bypasses, body.Revoke.Secret)
		}
		if body.Revoke == nil || body.Revoke.Regenerate {
			sequence++
			now := time.Now().UnixMilli()
			var expires *int64
			if body.TTL != nil {
				value := now/1000 + *body.TTL
				expires = &value
			}
			bypasses[fmt.Sprintf("secret-%d", sequence)] = client.ProtectionBypass{Scope: "shareable-link", CreatedAt: now, CreatedBy: "user_123", Expires: expires}
		}
		json.NewEncoder(w).Encode(map[string]any{"protectionBypass": bypasses})
	}))
	defer server.Close()
	c := client.New("abcdefghijklmnopqrstuvwx").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_123"})
	config := func(rotation string, ttl string) string {
		return fmt.Sprintf(`
resource "vercel_shareable_link" "test" {
 alias = "preview.vercel.app"
 %s
 rotation_id = %q
}
data "vercel_shareable_link" "test" {
 alias = vercel_shareable_link.test.alias
 depends_on = [vercel_shareable_link.test]
}
`, ttl, rotation)
	}
	testresource.Test(t, testresource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"vercel": providerserver.NewProtocol6WithError(&shareableTestProvider{client: c})},
		Steps: []testresource.TestStep{
			{Config: config("1", "ttl_seconds = 600"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("vercel_shareable_link.test", "secret", "secret-1"),
				testresource.TestCheckResourceAttr("vercel_shareable_link.test", "team_id", "team_123"),
				testresource.TestCheckResourceAttrPair("data.vercel_shareable_link.test", "secret", "vercel_shareable_link.test", "secret"),
			)},
			{Config: config("1", "ttl_seconds = 600"), PlanOnly: true},
			{ResourceName: "vercel_shareable_link.test", ImportState: true, ImportStateId: "team_123/preview.vercel.app", ImportStateVerify: true, ImportStateVerifyIgnore: []string{"rotation_id"}},
			{Config: config("2", "ttl_seconds = 600"), Check: testresource.TestCheckResourceAttr("vercel_shareable_link.test", "secret", "secret-2")},
			{Config: config("2", ""), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("vercel_shareable_link.test", "secret", "secret-3"),
				testresource.TestCheckNoResourceAttr("vercel_shareable_link.test", "expires_at"),
			)},
			{Config: config("2", ""), PlanOnly: true},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	if len(bypasses) != 1 || bypasses["*"].Scope != "alias-protection-override" {
		t.Fatalf("destroy affected unrelated bypasses or retained the link: %v", bypasses)
	}
}
