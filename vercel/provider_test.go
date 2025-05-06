package vercel_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/vercel/terraform-provider-vercel/v3/client"
	"github.com/vercel/terraform-provider-vercel/v3/vercel"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"vercel": providerserver.NewProtocol6WithError(vercel.New()),
}

var tc *client.Client

func testClient(t *testing.T) *client.Client {
	if tc == nil {
		tc = client.New(apiToken(t))
	}

	return tc
}

func apiToken(t *testing.T) string {
	value := os.Getenv("VERCEL_API_TOKEN")
	if value == "" {
		t.Fatalf("Missing required environment variable VERCEL_API_TOKEN")
	}
	return value
}

func testGithubRepo(t *testing.T) string {
	value := os.Getenv("VERCEL_TERRAFORM_TESTING_GITHUB_REPO")
	if value == "" {
		t.Fatalf("Missing required environment variable VERCEL_TERRAFORM_TESTING_GITHUB_REPO")
	}
	return value
}

func testBitbucketRepo(t *testing.T) string {
	value := os.Getenv("VERCEL_TERRAFORM_TESTING_BITBUCKET_REPO")
	if value == "" {
		t.Fatalf("Missing required environment variable VERCEL_TERRAFORM_TESTING_BITBUCKET_REPO")
	}
	return value
}

func testTeam(t *testing.T) string {
	value := os.Getenv("VERCEL_TERRAFORM_TESTING_TEAM")
	if value == "" {
		t.Fatalf("Missing required environment variable VERCEL_TERRAFORM_TESTING_TEAM")
	}
	return value
}

func teamIDConfig(t *testing.T) string {
	return fmt.Sprintf("team_id = \"%s\"", testTeam(t))
}

func testDomain(t *testing.T) string {
	value := os.Getenv("VERCEL_TERRAFORM_TESTING_DOMAIN")
	if value == "" {
		t.Fatalf("Missing required environment variable VERCEL_TERRAFORM_TESTING_DOMAIN")
	}
	return value
}

func testAdditionalUser(t *testing.T) string {
	value := os.Getenv("VERCEL_TERRAFORM_TESTING_ADDITIONAL_USER")
	if value == "" {
		t.Fatalf("Missing required environment variable VERCEL_TERRAFORM_TESTING_ADDITIONAL_USER")
	}
	return value
}

func testExistingIntegration(t *testing.T) string {
	value := os.Getenv("VERCEL_TERRAFORM_TESTING_EXISTING_INTEGRATION")
	if value == "" {
		t.Fatalf("Missing required environment variable VERCEL_TERRAFORM_TESTING_EXISTING_INTEGRATION")
	}
	return value
}
