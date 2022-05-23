package vercel_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/vercel/terraform-provider-vercel/client"
	"github.com/vercel/terraform-provider-vercel/vercel"
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
	mustHaveEnv(t, "VERCEL_TERRAFORM_TESTING_GITHUB_REPO")
	mustHaveEnv(t, "VERCEL_TERRAFORM_TESTING_GITLAB_REPO")
	mustHaveEnv(t, "VERCEL_TERRAFORM_TESTING_BITBUCKET_REPO")
	mustHaveEnv(t, "VERCEL_TERRAFORM_TESTING_TEAM")
	mustHaveEnv(t, "VERCEL_TERRAFORM_TESTING_DOMAIN")
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

func testGitlabRepo() string {
	return os.Getenv("VERCEL_TERRAFORM_TESTING_GITLAB_REPO")
}

func testBitbucketRepo() string {
	return os.Getenv("VERCEL_TERRAFORM_TESTING_BITBUCKET_REPO")
}

func testTeam() string {
	return os.Getenv("VERCEL_TERRAFORM_TESTING_TEAM")
}

func testDomain() string {
	return os.Getenv("VERCEL_TERRAFORM_TESTING_DOMAIN")
}
