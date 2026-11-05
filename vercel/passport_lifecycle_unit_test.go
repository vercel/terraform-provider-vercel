package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

type passportTestProvider struct {
	provider.Provider
	client *client.Client
}

func (p *passportTestProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	resp.ResourceData = p.client
	resp.DataSourceData = p.client
}

func TestPassportDisabledPlanAndApply(t *testing.T) {
	team := map[string]any{"id": "team_test", "name": "test", "slug": "test"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v2/teams/team_test" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL)
		}
		switch req.Method {
		case http.MethodPatch:
			var patch map[string]any
			if err := json.NewDecoder(req.Body).Decode(&patch); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			for key, value := range patch {
				team[key] = value
			}
		case http.MethodGet:
		default:
			t.Errorf("unexpected method: %s", req.Method)
		}
		if err := json.NewEncoder(w).Encode(team); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	factories := map[string]func() (tfprotov6.ProviderServer, error){"vercel": providerserver.NewProtocol6WithError(&passportTestProvider{
		Provider: New(), client: client.New("test").WithBaseURL(server.URL),
	})}
	config := func(passport string) string {
		return fmt.Sprintf(`
resource "vercel_team_config" "test" {
 id = "team_test"
 %s
}
data "vercel_team_config" "test" {
 id = vercel_team_config.test.id
 depends_on = [vercel_team_config.test]
}
`, passport)
	}
	disabled := config(`default_passport = { enabled = false }`)
	enabled := config(`default_passport = { connector_id = "scl_test", deployment_type = "preview" }`)
	checkDisabled := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr("vercel_team_config.test", "default_passport.enabled", "false"),
		resource.TestCheckNoResourceAttr("vercel_team_config.test", "default_passport.connector_id"),
		resource.TestCheckNoResourceAttr("vercel_team_config.test", "default_passport.deployment_type"),
		resource.TestCheckResourceAttr("data.vercel_team_config.test", "default_passport.enabled", "false"),
		resource.TestCheckNoResourceAttr("data.vercel_team_config.test", "default_passport.connector_id"),
		resource.TestCheckNoResourceAttr("data.vercel_team_config.test", "default_passport.deployment_type"),
	)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{Config: disabled, Check: checkDisabled},
			{Config: disabled, PlanOnly: true},
			{Config: enabled, Check: resource.TestCheckResourceAttr("vercel_team_config.test", "default_passport.deployment_type", "preview")},
			{Config: disabled, Check: checkDisabled},
			{Config: disabled, PlanOnly: true},
			{Config: config(""), Check: checkDisabled},
			{Config: config(`default_passport = { connector_id = "scl_test" }`), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("vercel_team_config.test", "default_passport.deployment_type", "all"),
				resource.TestCheckResourceAttr("data.vercel_team_config.test", "default_passport.deployment_type", "all"),
			)},
		},
	})
}
