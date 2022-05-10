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

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("VERCEL_API_TOKEN"); v == "" {
		t.Fatal("VERCEL_API_TOKEN must be set for acceptance tests")
	}
	if v := testTeam(); v == "" {
		t.Fatal("VERCEL_TERRAFORM_TESTING_TEAM must be set for acceptance tests against a specific team")
	}
	if v := testGithubRepo(); v == "" {
		t.Fatal("VERCEL_TERRAFORM_TESTING_GITHUB_REPO must be set for acceptance tests against a github repository")
	}
	if v := testGitlabRepo(); v == "" {
		t.Fatal("VERCEL_TERRAFORM_TESTING_GITLAB_REPO must be set for acceptance tests against a gitlab repository")
	}
	if v := testBitbucketRepo(); v == "" {
		t.Fatal("VERCEL_TERRAFORM_TESTING_BITBUCKET_REPO must be set for acceptance tests against a bitbucket repository")
	}
}

var tc *client.Client

func testClient() *client.Client {
	if tc == nil {
		tc = client.New(os.Getenv("VERCEL_API_TOKEN"))
	}

	return tc
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
