package vercel_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/vercel/terraform-provider-vercel/v2/client"
	"github.com/vercel/terraform-provider-vercel/v2/vercel"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"vercel": providerserver.NewProtocol6WithError(vercel.New()),
}

func mustHaveEnv(t *testing.T, name string) {
	if os.Getenv(name) == "" {
		t.Fatalf("%s environment variable must be set for acceptance tests", name)
	}
}

func testAccPreCheck(t *testing.T) {
	mustHaveEnv(t, "VERCEL_API_TOKEN")
	mustHaveEnv(t, "VERCEL_TERRAFORM_TESTING_TEAM")
	mustHaveEnv(t, "VERCEL_TERRAFORM_TESTING_GITHUB_REPO")
	mustHaveEnv(t, "VERCEL_TERRAFORM_TESTING_GITLAB_REPO")
	mustHaveEnv(t, "VERCEL_TERRAFORM_TESTING_BITBUCKET_REPO")
	mustHaveEnv(t, "VERCEL_TERRAFORM_TESTING_DOMAIN")
	mustHaveEnv(t, "VERCEL_TERRAFORM_TESTING_ADDITIONAL_USER")
	mustHaveEnv(t, "VERCEL_TERRAFORM_TESTING_EXISTING_INTEGRATION")
}

var tc *client.Client

func testClient() *client.Client {
	if tc == nil {
		tc = client.New(apiToken())
	}

	return tc
}

func apiToken() string {
	return os.Getenv("VERCEL_API_TOKEN")
}

func testGithubRepo() string {
	return os.Getenv("VERCEL_TERRAFORM_TESTING_GITHUB_REPO")
}

func testBitbucketRepo() string {
	return os.Getenv("VERCEL_TERRAFORM_TESTING_BITBUCKET_REPO")
}

func testTeam() string {
	return os.Getenv("VERCEL_TERRAFORM_TESTING_TEAM")
}

func teamIDConfig() string {
	if testTeam() == "" {
		return ""
	}
	return fmt.Sprintf("team_id = \"%s\"", testTeam())
}

func testDomain() string {
	return os.Getenv("VERCEL_TERRAFORM_TESTING_DOMAIN")
}

func testAdditionalUser() string {
	return os.Getenv("VERCEL_TERRAFORM_TESTING_ADDITIONAL_USER")
}

func testExistingIntegration() string {
	return os.Getenv("VERCEL_TERRAFORM_TESTING_EXISTING_INTEGRATION")
}
