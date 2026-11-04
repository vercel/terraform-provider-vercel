package vercel_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vercel/terraform-provider-vercel/v5/client"
	"github.com/vercel/terraform-provider-vercel/v5/vercel"
)

type protectionInventoryTestProvider struct {
	provider.Provider
	client *client.Client
}

func (p *protectionInventoryTestProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	resp.DataSourceData = p.client
}

func TestProtectionInventoryDataSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Query().Get("teamId") != "team_test" {
			t.Errorf("unexpected request %s %s", req.Method, req.URL)
		}
		switch req.URL.Path {
		case "/v9/projects/prj_test":
			fmt.Fprint(w, `{"id":"prj_test","accountId":"team_test","protectionBypass":{"private-automation-secret":{"scope":"automation-bypass","createdAt":1000}},"optionsAllowlist":{"paths":[{"value":"/api"}]},"trustedSources":{"projects":{"prj_test":{"customAllow":[]}},"enableVercelCiSameRepository":false}}`)
		case "/v4/aliases":
			fmt.Fprint(w, `{"aliases":[{"uid":"alias_test","alias":"preview.example","projectId":"prj_test","protectionBypass":{"private-shareable-secret":{"scope":"shareable-link","expires":1},"user_test":{"scope":"user","access":"granted"}}}],"pagination":{"next":null}}`)
		default:
			t.Errorf("unexpected API path %s", req.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	factories := map[string]func() (tfprotov6.ProviderServer, error){
		"vercel": providerserver.NewProtocol6WithError(&protectionInventoryTestProvider{
			Provider: vercel.New(), client: client.New("TOKEN").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_test"}),
		}),
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `data "vercel_project_protection_inventory" "test" { project_id = "prj_test" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_project_protection_inventory.test", "id", "prj_test"),
					resource.TestCheckResourceAttr("data.vercel_project_protection_inventory.test", "team_id", "team_test"),
					resource.TestCheckResourceAttr("data.vercel_project_protection_inventory.test", "complete", "false"),
					resource.TestCheckResourceAttr("data.vercel_project_protection_inventory.test", "entries.#", "5"),
					resource.TestCheckTypeSetElemNestedAttrs("data.vercel_project_protection_inventory.test", "entries.*", map[string]string{"scope": "trusted-project", "status": "disabled", "custom_allow.#": "0"}),
					resource.TestCheckTypeSetElemNestedAttrs("data.vercel_project_protection_inventory.test", "entries.*", map[string]string{"scope": "shareable-link", "status": "expired", "expires_at": "1"}),
					func(state *terraform.State) error {
						for _, res := range state.RootModule().Resources {
							for key, value := range res.Primary.Attributes {
								if strings.Contains(key+value, "private-automation-secret") || strings.Contains(key+value, "private-shareable-secret") {
									return fmt.Errorf("bearer credential leaked into Terraform state")
								}
							}
						}
						return nil
					},
				),
			},
			{
				Config: `data "vercel_project_protection_inventory" "test" {
  project_id = "prj_test"
  lifecycle {
    postcondition {
      condition = length(self.entries) == 0
      error_message = "Unapproved protection exceptions exist."
    }
  }
}`,
				PlanOnly: true, ExpectError: regexp.MustCompile("Unapproved protection exceptions exist"),
			},
			{
				Config: `data "vercel_project_protection_inventory" "test" {
  project_id = "prj_test"
  lifecycle {
    postcondition {
      condition = self.complete
      error_message = "Protection inventory is incomplete."
    }
  }
}`,
				PlanOnly: true, ExpectError: regexp.MustCompile("Protection inventory is incomplete"),
			},
		},
	})
}
