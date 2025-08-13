package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

func testCheckDrainExists(testClient *client.Client, teamID, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetDrain(context.Background(), rs.Primary.ID, teamID)
		return err
	}
}

func testCheckDrainDeleted(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetDrain(context.Background(), rs.Primary.ID, teamID)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("Unexpected error checking for deleted drain: %s", err)
		}

		return nil
	}
}

func TestAcc_DrainResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckDrainDeleted(testClient(t), "vercel_drain.minimal", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceDrain(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckDrainExists(testClient(t), testTeam(t), "vercel_drain.minimal"),
					resource.TestCheckResourceAttr("vercel_drain.minimal", "name", "minimal-drain"),
					resource.TestCheckResourceAttr("vercel_drain.minimal", "projects", "all"),
					resource.TestCheckResourceAttr("vercel_drain.minimal", "schemas.%", "1"),
					resource.TestCheckResourceAttr("vercel_drain.minimal", "schemas.log.version", "1"),
					resource.TestCheckResourceAttr("vercel_drain.minimal", "delivery.type", "http"),
					resource.TestCheckResourceAttr("vercel_drain.minimal", "delivery.encoding", "json"),
					resource.TestCheckResourceAttr("vercel_drain.minimal", "delivery.endpoint.url", "https://example.com/webhook"),
					resource.TestCheckResourceAttrSet("vercel_drain.minimal", "id"),
					resource.TestCheckResourceAttrSet("vercel_drain.minimal", "team_id"),
					resource.TestCheckResourceAttrSet("vercel_drain.minimal", "status"),

					testCheckDrainExists(testClient(t), testTeam(t), "vercel_drain.maximal"),
					resource.TestCheckResourceAttr("vercel_drain.maximal", "name", "maximal-drain"),
					resource.TestCheckResourceAttr("vercel_drain.maximal", "projects", "some"),
					resource.TestCheckResourceAttr("vercel_drain.maximal", "project_ids.#", "1"),
					resource.TestCheckResourceAttr("vercel_drain.maximal", "schemas.%", "2"),
					resource.TestCheckResourceAttr("vercel_drain.maximal", "schemas.log.version", "1"),
					resource.TestCheckResourceAttr("vercel_drain.maximal", "schemas.trace.version", "1"),
					resource.TestCheckResourceAttr("vercel_drain.maximal", "delivery.type", "http"),
					resource.TestCheckResourceAttr("vercel_drain.maximal", "delivery.encoding", "ndjson"),
					resource.TestCheckResourceAttr("vercel_drain.maximal", "delivery.endpoint.url", "https://example.com/drain"),
					resource.TestCheckResourceAttr("vercel_drain.maximal", "delivery.compression", "gzip"),
					resource.TestCheckResourceAttr("vercel_drain.maximal", "delivery.headers.%", "2"),
					resource.TestCheckResourceAttr("vercel_drain.maximal", "delivery.headers.Authorization", "Bearer token123"),
					resource.TestCheckResourceAttr("vercel_drain.maximal", "delivery.headers.Content-Type", "application/x-ndjson"),
					resource.TestCheckResourceAttr("vercel_drain.maximal", "sampling.#", "2"),
					resource.TestCheckResourceAttrSet("vercel_drain.maximal", "id"),
					resource.TestCheckResourceAttrSet("vercel_drain.maximal", "team_id"),
					resource.TestCheckResourceAttrSet("vercel_drain.maximal", "status"),
				),
			},
		},
	})
}

func TestAcc_DrainResourceUpdate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckDrainDeleted(testClient(t), "vercel_drain.update_test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceDrainInitial()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckDrainExists(testClient(t), testTeam(t), "vercel_drain.update_test"),
					resource.TestCheckResourceAttr("vercel_drain.update_test", "name", "initial-name"),
					resource.TestCheckResourceAttr("vercel_drain.update_test", "delivery.encoding", "json"),
				),
			},
			{
				Config: cfg(testAccResourceDrainUpdated()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckDrainExists(testClient(t), testTeam(t), "vercel_drain.update_test"),
					resource.TestCheckResourceAttr("vercel_drain.update_test", "name", "updated-name"),
					resource.TestCheckResourceAttr("vercel_drain.update_test", "delivery.encoding", "ndjson"),
				),
			},
		},
	})
}

func TestAcc_DrainDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccDataSourceDrain()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_drain.test", "name", "data-source-test"),
					resource.TestCheckResourceAttr("data.vercel_drain.test", "projects", "all"),
					resource.TestCheckResourceAttr("data.vercel_drain.test", "delivery.type", "http"),
					resource.TestCheckResourceAttr("data.vercel_drain.test", "delivery.encoding", "json"),
					resource.TestCheckResourceAttr("data.vercel_drain.test", "delivery.endpoint.url", "https://example.com/webhook"),
					resource.TestCheckResourceAttrSet("data.vercel_drain.test", "id"),
					resource.TestCheckResourceAttrSet("data.vercel_drain.test", "team_id"),
					resource.TestCheckResourceAttrSet("data.vercel_drain.test", "status"),
				),
			},
		},
	})
}

func TestAcc_DrainResourceOTLP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckDrainDeleted(testClient(t), "vercel_drain.otlp_test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceDrainOTLP()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckDrainExists(testClient(t), testTeam(t), "vercel_drain.otlp_test"),
					resource.TestCheckResourceAttr("vercel_drain.otlp_test", "name", "otlp-drain"),
					resource.TestCheckResourceAttr("vercel_drain.otlp_test", "delivery.type", "otlphttp"),
					resource.TestCheckResourceAttr("vercel_drain.otlp_test", "delivery.endpoint.traces", "https://otlp.example.com/v1/traces"),
					resource.TestCheckResourceAttr("vercel_drain.otlp_test", "delivery.encoding", "proto"),
				),
			},
		},
	})
}

func TestAcc_DrainDataSourceOTLP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccDataSourceDrainOTLP()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_drain.otlp", "name", "otlp-data-source"),
					resource.TestCheckResourceAttr("data.vercel_drain.otlp", "delivery.type", "otlphttp"),
					resource.TestCheckResourceAttr("data.vercel_drain.otlp", "delivery.endpoint.traces", "https://otlp.example.com/v1/traces"),
					resource.TestCheckResourceAttr("data.vercel_drain.otlp", "delivery.encoding", "proto"),
				),
			},
		},
	})
}

func testAccResourceDrain(name string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
    name = "test-acc-%[1]s"
}

resource "vercel_drain" "minimal" {
    name     = "minimal-drain"
    projects = "all"
    schemas = {
        log = {
            version = "1"
        }
    }
    delivery = {
        type     = "http"
        endpoint = {
            url = "https://example.com/webhook"
        }
        encoding = "json"
        headers  = {}
    }
}

resource "vercel_drain" "maximal" {
    name        = "maximal-drain"
    projects    = "some"
    project_ids = [vercel_project.test.id]
    filter      = "level >= 'info'"
    schemas = {
        log = {
            version = "1"
        }
        trace = {
            version = "1"
        }
    }
    delivery = {
        type        = "http"
        endpoint = {
            url = "https://example.com/drain"
        }
        encoding    = "ndjson"
        compression = "gzip"
        headers = {
            "Authorization" = "Bearer token123"
            "Content-Type"  = "application/x-ndjson"
        }
        secret = "secret123"
    }
    sampling = [
        {
            type        = "head_sampling"
            rate        = 0.1
            environment = "production"
        },
        {
            type         = "head_sampling"
            rate         = 0.5
            environment  = "preview"
            request_path = "/api/"
        }
    ]
    transforms = [
        {
            id = "transform1"
        }
    ]
}
`, name)
}

func testAccResourceDrainInitial() string {
	return `
resource "vercel_drain" "update_test" {
    name     = "initial-name"
    projects = "all"
    schemas = {
        log = {
            version = "1"
        }
    }
    delivery = {
        type     = "http"
        endpoint = {
            url = "https://example.com/webhook"
        }
        encoding = "json"
        headers  = {}
    }
}`
}

func testAccResourceDrainUpdated() string {
	return `
resource "vercel_drain" "update_test" {
    name     = "updated-name"
    projects = "all"
    schemas = {
        log = {
            version = "1"
        }
    }
    delivery = {
        type     = "http"
        endpoint = {
            url = "https://example.com/webhook"
        }
        encoding = "ndjson"
        headers  = {}
    }
}`
}

func testAccDataSourceDrain() string {
	return `
resource "vercel_drain" "for_data_source" {
    name     = "data-source-test"
    projects = "all"
    schemas = {
        log = {
            version = "1"
        }
    }
    delivery = {
        type     = "http"
        endpoint = {
            url = "https://example.com/webhook"
        }
        encoding = "json"
        headers  = {}
    }
}

data "vercel_drain" "test" {
    id = vercel_drain.for_data_source.id
}`
}

func testAccResourceDrainOTLP() string {
	return `
resource "vercel_drain" "otlp_test" {
    name     = "otlp-drain"
    projects = "all"
    schemas = {
        trace = {
            version = "1"
        }
    }
    delivery = {
        type     = "otlphttp"
        endpoint = {
            traces = "https://otlp.example.com/v1/traces"
        }
        encoding = "proto"
        headers  = {}
    }
}`
}

func testAccDataSourceDrainOTLP() string {
	return `
resource "vercel_drain" "for_otlp_data_source" {
    name     = "otlp-data-source"
    projects = "all"
    schemas = {
        trace = {
            version = "1"
        }
    }
    delivery = {
        type     = "otlphttp"
        endpoint = {
            traces = "https://otlp.example.com/v1/traces"
        }
        encoding = "proto"
        headers  = {}
    }
}

data "vercel_drain" "otlp" {
    id = vercel_drain.for_otlp_data_source.id
}`
}
