package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

type organisationTeamMockProvider struct {
	provider.Provider
	client *client.Client
}

func (p *organisationTeamMockProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	resp.ResourceData = p.client
}

func TestOrganisationTeamLifecycle(t *testing.T) {
	var mu sync.Mutex
	name, slug := "Product", "acme-product"
	created, deleted, updated := false, false, false
	reads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.URL.RawQuery != "" {
			t.Errorf("unexpected query scope: %s", r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "POST /v1/organizations/org_1/teams":
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body["mode"] != "create" || body["name"] != name || body["slug"] != slug || body["billingPlan"] != "platform" {
				t.Errorf("unexpected create: %#v", body)
			}
			created = true
			fmt.Fprint(w, `{"organizationId":"org_1","teamId":"team_1","billingPlan":"platform"}`)
		case "GET /v1/organizations/org_1":
			fmt.Fprint(w, `{"teamId":"team_root"}`)
		case "GET /v1/organizations/org_1/teams":
			reads++
			if reads == 1 {
				fmt.Fprint(w, `{"teams":[]}`)
				return
			}
			if !created || deleted {
				fmt.Fprint(w, `{"teams":[]}`)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"teams": []client.OrganisationTeam{{OrganisationID: "org_1", TeamID: "team_1", Name: name, Slug: slug, BillingPlan: "platform"}}})
		case "PATCH /v2/teams/team_1":
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if len(body) != 2 {
				t.Errorf("unexpected update fields: %#v", body)
			}
			name, slug = body["name"], body["slug"]
			updated = true
			json.NewEncoder(w).Encode(map[string]string{"id": "team_1", "name": name, "slug": slug})
		case "DELETE /v1/organizations/org_1/teams/team_1":
			deleted = true
			fmt.Fprint(w, `{"organizationId":"org_1","teamId":"team_1"}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			http.Error(w, "unexpected request", 400)
		}
	}))
	defer server.Close()
	factories := map[string]func() (tfprotov6.ProviderServer, error){"vercel": providerserver.NewProtocol6WithError(&organisationTeamMockProvider{
		Provider: New(), client: client.New("TOKEN").WithBaseURL(server.URL).WithTeam(client.Team{ID: "unrelated_provider_team"}),
	})}
	config := func(name, slug string) string {
		return fmt.Sprintf(`
resource "vercel_organisation_team" "test" {
 organisation_id = "org_1"
 name = %q
 slug = %q
 billing_plan = "platform"
}
`, name, slug)
	}
	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: factories, Steps: []tfresource.TestStep{
		{Config: config("Product", "acme-product"), Check: tfresource.ComposeTestCheckFunc(
			tfresource.TestCheckResourceAttr("vercel_organisation_team.test", "id", "team_1"),
			tfresource.TestCheckResourceAttr("vercel_organisation_team.test", "organisation_id", "org_1"),
		)},
		{Config: config("Renamed", "acme-renamed")},
		{ResourceName: "vercel_organisation_team.test", ImportState: true, ImportStateId: "org_1/team_1", ImportStateVerify: true},
	}})
	mu.Lock()
	defer mu.Unlock()
	if !created || !updated || !deleted {
		t.Fatalf("create=%v update=%v delete=%v", created, updated, deleted)
	}
}

func TestOrganisationTeamReadAndDeleteFailures(t *testing.T) {
	for _, status := range []int{400, 403, 404, 409, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				fmt.Fprint(w, `{"error":{"code":"team_cannot_be_deleted","message":"failure"}}`)
			}))
			defer server.Close()
			ctx := context.Background()
			r := &organisationTeamResource{client: client.New("TOKEN").WithBaseURL(server.URL)}
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			state := tfsdk.State{Schema: schemaResp.Schema}
			diags := state.Set(ctx, &organisationTeam{ID: types.StringValue("team_1"), OrganisationID: types.StringValue("org_1"), Name: types.StringValue("Product"), Slug: types.StringValue("product"), BillingPlan: types.StringValue("enterprise")})
			if diags.HasError() {
				t.Fatal(diags)
			}
			read := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &read)
			if status == 404 {
				if read.Diagnostics.HasError() || !read.State.Raw.IsNull() {
					t.Fatalf("read: %v", read.Diagnostics)
				}
			} else if !read.Diagnostics.HasError() || !read.State.Raw.Equal(state.Raw) {
				t.Fatalf("read lost state: %v", read.Diagnostics)
			}
			var deleted resource.DeleteResponse
			r.Delete(ctx, resource.DeleteRequest{State: state}, &deleted)
			if deleted.Diagnostics.HasError() != (status != 404) {
				t.Fatalf("delete: %v", deleted.Diagnostics)
			}
		})
	}
}

func TestOrganisationTeamValidation(t *testing.T) {
	ctx := context.Background()
	var response resource.SchemaResponse
	newOrganisationTeamResource().Schema(ctx, resource.SchemaRequest{}, &response)
	for _, tc := range []struct {
		attribute, value string
		valid            bool
	}{
		{"name", "Product", true}, {"name", "Product Team", true}, {"name", " Product ", false}, {"name", " ", false},
		{"slug", "acme-product", true}, {"slug", "Product", false}, {"slug", " product ", false}, {"slug", "-product", false},
		{"billing_plan", "platform", true}, {"billing_plan", "enterprise", true}, {"billing_plan", "pro", false},
		{"organisation_id", "", false},
	} {
		t.Run(tc.attribute+"="+tc.value, func(t *testing.T) {
			var result validator.StringResponse
			for _, v := range response.Schema.Attributes[tc.attribute].(schema.StringAttribute).Validators {
				v.ValidateString(ctx, validator.StringRequest{Path: path.Root(tc.attribute), ConfigValue: types.StringValue(tc.value)}, &result)
			}
			if result.Diagnostics.HasError() == tc.valid {
				t.Fatalf("valid=%v, diagnostics=%v", tc.valid, result.Diagnostics)
			}
		})
	}
}

func TestOrganisationTeamImportRejectsRootAndMalformedIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"teamId":"team_root"}`)
	}))
	defer server.Close()
	r := &organisationTeamResource{client: client.New("TOKEN").WithBaseURL(server.URL)}
	for _, id := range []string{"org_1/team_root", "team_1", "org_1/", "/team_1", "org_1/team_1/extra"} {
		t.Run(id, func(t *testing.T) {
			var response resource.ImportStateResponse
			r.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, &response)
			if !response.Diagnostics.HasError() {
				t.Fatal("invalid import accepted")
			}
		})
	}
}
