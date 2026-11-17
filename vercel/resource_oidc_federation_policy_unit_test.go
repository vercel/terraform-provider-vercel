package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

type oidcFederationPolicyMockProvider struct {
	provider.Provider
	client *client.Client
}

func (p *oidcFederationPolicyMockProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	resp.ResourceData = p.client
}

func TestOIDCFederationClientMapping(t *testing.T) {
	for clientName, clientID := range map[string]string{
		"turborepo": "cl_kyUx2zVvA4MGptBohkmtYHJly2XltXzD",
		"vercel":    "cl_HYyOPBNtFMfHhaUn9L4QPfTZz6TP47bp",
		"vcr":       "cl_inrfNy8noLlhRrGbPEm0z47woXNcJVZ0",
	} {
		if got := oidcFederationClientID(clientName); got != clientID {
			t.Errorf("oidcFederationClientID(%q) = %q, want %q", clientName, got, clientID)
		}
		if got, ok := oidcFederationClientName(clientID); !ok || got != clientName {
			t.Errorf("oidcFederationClientName(%q) = %q, %v, want %q", clientID, got, ok, clientName)
		}
	}
	if _, ok := oidcFederationClientName("cl_unknown"); ok {
		t.Error("an unknown client ID mapped to a client")
	}
}

// oidcFederationPolicyAPI is the OIDC federation policy API for one team, holding at most one policy.
func oidcFederationPolicyAPI(t *testing.T) (*httptest.Server, func() (created, updated, deleted map[string]any)) {
	var mu sync.Mutex
	var policy, created, updated map[string]any
	deleted := map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.URL.Query().Get("teamId") != "team_1" {
			t.Errorf("unexpected team scope: %s", r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "POST /v1/oauth-apps/oidc-federation/policies":
			if err := json.NewDecoder(r.Body).Decode(&created); err != nil {
				t.Error(err)
			}
			policy = map[string]any{"policyId": "pol_1", "teamId": "team_1"}
			for key, value := range created {
				policy[key] = value
			}
			json.NewEncoder(w).Encode(policy)
		case "GET /v1/oauth-apps/oidc-federation/policies":
			policies := []map[string]any{}
			if policy != nil {
				policies = append(policies, policy)
			}
			json.NewEncoder(w).Encode(map[string]any{"policies": policies})
		case "PATCH /v1/oauth-apps/oidc-federation/policies/pol_1":
			if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
				t.Error(err)
			}
			for key, value := range updated {
				policy[key] = value
			}
			json.NewEncoder(w).Encode(policy)
		case "DELETE /v1/oauth-apps/oidc-federation/policies/pol_1":
			deleted = policy
			policy = nil
			fmt.Fprint(w, `{}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			http.Error(w, "unexpected request", 400)
		}
	}))
	return server, func() (map[string]any, map[string]any, map[string]any) {
		mu.Lock()
		defer mu.Unlock()
		return created, updated, deleted
	}
}

func oidcFederationPolicyFactories(server *httptest.Server) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){"vercel": providerserver.NewProtocol6WithError(&oidcFederationPolicyMockProvider{
		Provider: New(), client: client.New("TOKEN").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_1"}),
	})}
}

func oidcFederationPolicyConfig(clientName, name, access string) string {
	return fmt.Sprintf(`
resource "vercel_oidc_federation_policy" "test" {
  name       = %[2]q
  client     = %[1]q
  issuer_url = "https://token.actions.githubusercontent.com"

  claims = [
    {
      name   = "aud"
      values = [{ value = "https://vercel.com/acme" }]
    },
    {
      name   = "repository"
      values = [{ value = "acme/widgets" }]
    }
  ]

  %[3]s

  resources = {
    project_ids = ["prj_1"]
  }
}
`, clientName, name, access)
}

func TestOIDCFederationPolicyVCRLifecycle(t *testing.T) {
	server, requests := oidcFederationPolicyAPI(t)
	defer server.Close()
	const resourceName = "vercel_oidc_federation_policy.test"
	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: oidcFederationPolicyFactories(server), Steps: []tfresource.TestStep{
		{
			Config: oidcFederationPolicyConfig("vcr", "Push images", `permissions = ["read-write:vcr"]`),
			Check: tfresource.ComposeAggregateTestCheckFunc(
				tfresource.TestCheckResourceAttr(resourceName, "id", "pol_1"),
				tfresource.TestCheckResourceAttr(resourceName, "client", "vcr"),
				tfresource.TestCheckTypeSetElemAttr(resourceName, "permissions.*", "read-write:vcr"),
				tfresource.TestCheckNoResourceAttr(resourceName, "commands"),
			),
		},
		{
			Config: oidcFederationPolicyConfig("vcr", "Pull images", `permissions = ["read:vcr"]`),
			Check: tfresource.ComposeAggregateTestCheckFunc(
				tfresource.TestCheckResourceAttr(resourceName, "client", "vcr"),
				tfresource.TestCheckTypeSetElemAttr(resourceName, "permissions.*", "read:vcr"),
			),
		},
		{ResourceName: resourceName, ImportState: true, ImportStateId: "team_1/pol_1", ImportStateVerify: true},
	}})

	created, updated, deleted := requests()
	if created["clientId"] != "cl_inrfNy8noLlhRrGbPEm0z47woXNcJVZ0" {
		t.Errorf("created with client ID %v", created["clientId"])
	}
	if _, ok := created["commands"]; ok {
		t.Errorf("created with commands: %#v", created)
	}
	if fmt.Sprint(created["permissions"]) != "[read-write:vcr]" || fmt.Sprint(updated["permissions"]) != "[read:vcr]" || updated["name"] != "Pull images" {
		t.Errorf("unexpected create %#v or update %#v", created, updated)
	}
	if deleted["policyId"] != "pol_1" {
		t.Errorf("the policy wasn't deleted: %#v", deleted)
	}
}

func TestOIDCFederationPolicyVCRValidation(t *testing.T) {
	server, _ := oidcFederationPolicyAPI(t)
	defer server.Close()
	for name, tc := range map[string]struct {
		access string
		err    string
	}{
		"wildcard":                 {`permissions = ["*"]`, `The .vcr. client accepts only .read:vcr. and .read-write:vcr.\s+permissions,\s+not "\*"`},
		"another app's permission": {`permissions = ["read-write:remote-cache"]`, `not\s+"read-write:remote-cache"`},
		"a permission beside one":  {`permissions = ["read:vcr", "read-write:project"]`, `not\s+"read-write:project"`},
		"commands":                 {`commands = ["deploy"]`, `.commands. is only supported for the .vercel. client`},
	} {
		t.Run(name, func(t *testing.T) {
			tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: oidcFederationPolicyFactories(server), Steps: []tfresource.TestStep{{
				Config:      oidcFederationPolicyConfig("vcr", "Push images", tc.access),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(tc.err),
			}}})
		})
	}
}

func TestOIDCFederationPolicyOtherClientsKeepTheirRules(t *testing.T) {
	server, _ := oidcFederationPolicyAPI(t)
	defer server.Close()
	for _, tc := range []struct{ clientName, access string }{
		{"turborepo", `permissions = ["*"]`},
		{"vercel", `permissions = ["*"]`},
		{"vercel", `commands = ["deploy"]`},
	} {
		t.Run(tc.clientName+" "+tc.access, func(t *testing.T) {
			tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: oidcFederationPolicyFactories(server), Steps: []tfresource.TestStep{{
				Config:             oidcFederationPolicyConfig(tc.clientName, "Other client", tc.access),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			}}})
		})
	}
}
