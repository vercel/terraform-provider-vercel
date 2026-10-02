package vercel_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

// The fixture uses dummy credentials and explicit endpoints. These tests verify
// configuration persistence, not an identity provider's visitor sign-in flow.
func testAccPassportApplication(t *testing.T) client.ConnectApplication {
	t.Helper()
	if os.Getenv("TF_ACC") != "true" {
		t.Skip("live Passport tests require TF_ACC=true")
	}
	name := "test-acc-passport-" + acctest.RandString(12)
	input := map[string]any{
		"type": "oauth", "name": name, "uid": "oauth/" + name,
		"data": map[string]any{
			"clientId": name, "clientSecret": "dummy-acceptance-test-secret",
			"serverConfig": map[string]any{
				"issuer":                 "https://accounts.google.com",
				"authorization_endpoint": "https://accounts.google.com/o/oauth2/v2/auth",
				"token_endpoint":         "https://oauth2.googleapis.com/token",
				"jwks_uri":               "https://www.googleapis.com/oauth2/v3/certs",
			},
			"userAuthorization": map[string]any{"enabled": true, "scopes": []string{"openid", "email", "profile"}},
		},
	}
	var application client.ConnectApplication
	passportFixtureRequest(t, http.MethodPost, "/v1/connect/connectors", input, &application)
	if application.ID == "" {
		t.Fatal("created Connect application is missing ID")
	}
	t.Cleanup(func() {
		passportFixtureRequest(t, http.MethodDelete, "/v1/connect/connectors/"+url.PathEscape(application.ID), nil, nil)
	})
	if application.UID == "" {
		t.Fatal("created Connect application is missing UID")
	}
	return application
}

func passportFixtureRequest(t *testing.T, method, path string, input, output any) {
	t.Helper()
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, "https://api.vercel.com"+path+"?teamId="+url.QueryEscape(testTeam(t)), bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+apiToken(t))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if method == http.MethodDelete && resp.StatusCode == http.StatusNotFound {
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var failure struct {
			Error struct {
				Code    string
				Message string
			}
		}
		_ = json.NewDecoder(resp.Body).Decode(&failure)
		t.Fatalf("Passport fixture %s failed (%d): %s: %s", method, resp.StatusCode, failure.Error.Code, failure.Error.Message)
	}
	if output != nil {
		if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAcc_PassportProject(t *testing.T) {
	application := testAccPassportApplication(t)
	name := "test-acc-passport-" + acctest.RandString(12)
	var projectID string
	config := func(passport, suffix string) string {
		return cfg(fmt.Sprintf(`
data "vercel_connect_application" "by_uid" { uid = %[1]q }
data "vercel_connect_application" "by_id" { id = %[2]q }
resource "vercel_project" "passport" {
 name = %[3]q
 %[4]s
}
data "vercel_project" "passport" {
 name = vercel_project.passport.name
 depends_on = [vercel_project.passport]
}
%[5]s
`, application.UID, application.ID, name, passport, suffix))
	}
	enabled := `passport = { connector_id = data.vercel_connect_application.by_uid.id }`
	preview := `passport = { connector_id = data.vercel_connect_application.by_uid.id, deployment_type = "preview" }`
	checkEnabled := func(scope string) resource.TestCheckFunc {
		return resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("vercel_project.passport", "passport.enabled", "true"),
			resource.TestCheckResourceAttr("vercel_project.passport", "passport.connector_id", application.ID),
			resource.TestCheckResourceAttr("vercel_project.passport", "passport.deployment_type", scope),
			resource.TestCheckResourceAttr("data.vercel_project.passport", "passport.deployment_type", scope),
			resource.TestCheckResourceAttrPair("data.vercel_connect_application.by_id", "id", "data.vercel_connect_application.by_uid", "id"),
			resource.TestCheckResourceAttr("data.vercel_connect_application.by_uid", "type", "oauth"),
			func(state *terraform.State) error {
				projectID = state.RootModule().Resources["vercel_project.passport"].Primary.ID
				return nil
			},
		)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.passport", testTeam(t)),
		Steps: []resource.TestStep{
			{Config: config(enabled, ""), Check: checkEnabled("all")},
			{Config: config(enabled, ""), PlanOnly: true},
			{Config: config(preview, ""), Check: checkEnabled("preview")},
			{ResourceName: "vercel_project.passport", ImportState: true, ImportStateVerify: true, ImportStateIdFunc: func(*terraform.State) (string, error) { return testTeam(t) + "/" + projectID, nil }},
			{PreConfig: func() {
				passportFixtureRequest(t, http.MethodPatch, "/v9/projects/"+url.PathEscape(projectID), map[string]any{"passport": nil}, nil)
			}, Config: config(preview, ""), Check: checkEnabled("preview")},
			// Removing the optional/computed block preserves the observed settings.
			{Config: config("", ""), Check: checkEnabled("preview")},
			{Config: config(`passport = { enabled = false }`, ""), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("vercel_project.passport", "passport.enabled", "false"),
				resource.TestCheckResourceAttr("data.vercel_project.passport", "passport.enabled", "false"),
				resource.TestCheckNoResourceAttr("vercel_project.passport", "passport.connector_id"),
			)},
			{Config: config(`passport = { enabled = false }`, ""), PlanOnly: true},
		},
	})
}

func TestAcc_PassportTeamDefault(t *testing.T) {
	// A separate team keeps this team-wide default change out of concurrent CI
	// acceptance shards. Local focused runs may explicitly select an idle test team.
	teamID := os.Getenv("VERCEL_TERRAFORM_TESTING_PASSPORT_TEAM")
	if teamID == "" {
		t.Skip("set VERCEL_TERRAFORM_TESTING_PASSPORT_TEAM to an isolated team for default tests")
	}
	t.Setenv("VERCEL_TERRAFORM_TESTING_TEAM", teamID)
	application := testAccPassportApplication(t)
	ctx := context.Background()
	team, err := testClient(t).GetTeam(ctx, testTeam(t))
	if err != nil {
		t.Fatal(err)
	}
	originalDefault := team.DefaultPassport
	t.Cleanup(func() {
		_, err := testClient(t).UpdateTeam(ctx, client.UpdateTeamRequest{TeamID: testTeam(t), DefaultPassport: &originalDefault})
		if err != nil {
			t.Errorf("restore original Passport default: %s", err)
		}
	})
	newProject := func() client.ProjectResponse {
		project, err := testClient(t).CreateProject(ctx, testTeam(t), client.CreateProjectRequest{Name: "test-acc-passport-" + acctest.RandString(12)})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := testClient(t).DeleteProject(ctx, project.ID, testTeam(t)); err != nil {
				t.Errorf("delete fixture project: %s", err)
			}
		})
		return project
	}
	existing := newProject()
	var inherited client.ProjectResponse
	config := func(passport string) string {
		setting := ""
		if passport != "" {
			setting = "default_passport = " + passport
		}
		return cfg(fmt.Sprintf(`
data "vercel_connect_application" "passport" { uid = %[1]q }
resource "vercel_team_config" "passport" {
 id = %[2]q
 %[3]s
}
data "vercel_team_config" "passport" {
 id = vercel_team_config.passport.id
 depends_on = [vercel_team_config.passport]
}
`, application.UID, testTeam(t), setting))
	}
	all := `{ connector_id = data.vercel_connect_application.passport.id }`
	preview := `{ connector_id = data.vercel_connect_application.passport.id, deployment_type = "preview" }`
	checkDefault := func(scope string) resource.TestCheckFunc {
		return resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("vercel_team_config.passport", "default_passport.connector_id", application.ID),
			resource.TestCheckResourceAttr("vercel_team_config.passport", "default_passport.deployment_type", scope),
			resource.TestCheckResourceAttr("data.vercel_team_config.passport", "default_passport.deployment_type", scope),
		)
	}
	checkUnchanged := func(project client.ProjectResponse) error {
		current, err := testClient(t).GetProject(ctx, project.ID, testTeam(t))
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(current.Passport, project.Passport) {
			return fmt.Errorf("team default changed an existing project's Passport settings")
		}
		return nil
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config(all), Check: resource.ComposeAggregateTestCheckFunc(checkDefault("all"), func(*terraform.State) error {
				if err := checkUnchanged(existing); err != nil {
					return err
				}
				inherited = newProject()
				if inherited.Passport == nil || inherited.Passport.ConnectorID != application.ID || inherited.Passport.DeploymentType != "all" {
					return fmt.Errorf("new project did not copy team Passport default")
				}
				return nil
			})},
			{ResourceName: "vercel_team_config.passport", ImportState: true, ImportStateVerify: true, ImportStateId: testTeam(t), ImportStateVerifyIgnore: []string{"avatar"}},
			{Config: config(preview), Check: resource.ComposeAggregateTestCheckFunc(checkDefault("preview"), func(*terraform.State) error { return checkUnchanged(inherited) })},
			{Config: config(""), Check: checkDefault("preview")},
			{Config: config(""), PlanOnly: true},
			{Config: config(`{ enabled = false }`), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("vercel_team_config.passport", "default_passport.enabled", "false"),
				resource.TestCheckResourceAttr("data.vercel_team_config.passport", "default_passport.enabled", "false"),
				func(*terraform.State) error { return checkUnchanged(inherited) },
			)},
			{Config: config(`{ enabled = false }`), PlanOnly: true},
		},
	})
}
