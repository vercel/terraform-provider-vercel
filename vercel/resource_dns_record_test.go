package vercel_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/vercel/terraform-provider-vercel/client"
)

func testAccDNSRecordDestroy(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient().GetDNSRecord(context.TODO(), rs.Primary.ID, teamID)

		var apiErr client.APIError
		if err == nil {
			return fmt.Errorf("Found project but expected it to have been deleted")
		}
		if err != nil && errors.As(err, &apiErr) {
			if apiErr.StatusCode == 404 {
				return nil
			}
			return fmt.Errorf("Unexpected error checking for deleted project: %s", apiErr)
		}

		return err
	}
}

func testAccDNSRecordExists(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient().GetDNSRecord(context.TODO(), rs.Primary.ID, teamID)
		return err
	}
}

func TestAcc_DNSRecord(t *testing.T) {
	t.Parallel()
	nameSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccDNSRecordDestroy("vercel_dns_record.a_without_ttl", ""),
			testAccDNSRecordDestroy("vercel_dns_record.a", ""),
			testAccDNSRecordDestroy("vercel_dns_record.aaaa", ""),
			testAccDNSRecordDestroy("vercel_dns_record.alias", ""),
			testAccDNSRecordDestroy("vercel_dns_record.caa", ""),
			testAccDNSRecordDestroy("vercel_dns_record.cname", ""),
			testAccDNSRecordDestroy("vercel_dns_record.mx", ""),
			testAccDNSRecordDestroy("vercel_dns_record.srv", ""),
			testAccDNSRecordDestroy("vercel_dns_record.txt", ""),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordConfig(testDomain(), nameSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDNSRecordExists("vercel_dns_record.a_without_ttl", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.a", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.a", "type", "A"),
					resource.TestCheckResourceAttr("vercel_dns_record.a", "ttl", "60"),
					resource.TestCheckResourceAttr("vercel_dns_record.a", "value", "127.0.0.1"),
					testAccDNSRecordExists("vercel_dns_record.a", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.a", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.a", "type", "A"),
					resource.TestCheckResourceAttr("vercel_dns_record.a", "ttl", "120"),
					resource.TestCheckResourceAttr("vercel_dns_record.a", "value", "127.0.0.1"),
					testAccDNSRecordExists("vercel_dns_record.aaaa", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.aaaa", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.aaaa", "type", "AAAA"),
					resource.TestCheckResourceAttr("vercel_dns_record.aaaa", "ttl", "120"),
					resource.TestCheckResourceAttr("vercel_dns_record.aaaa", "value", "::1"),
					testAccDNSRecordExists("vercel_dns_record.alias", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.alias", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.alias", "type", "ALIAS"),
					resource.TestCheckResourceAttr("vercel_dns_record.alias", "ttl", "120"),
					resource.TestCheckResourceAttr("vercel_dns_record.alias", "value", "example.com."),
					testAccDNSRecordExists("vercel_dns_record.caa", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.caa", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.caa", "type", "CAA"),
					resource.TestCheckResourceAttr("vercel_dns_record.caa", "ttl", "120"),
					resource.TestCheckResourceAttr("vercel_dns_record.caa", "value", "0 issue \"letsencrypt.org\""),
					testAccDNSRecordExists("vercel_dns_record.cname", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.cname", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.cname", "type", "CNAME"),
					resource.TestCheckResourceAttr("vercel_dns_record.cname", "ttl", "120"),
					resource.TestCheckResourceAttr("vercel_dns_record.cname", "value", "example.com."),
					testAccDNSRecordExists("vercel_dns_record.mx", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.mx", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.mx", "type", "MX"),
					resource.TestCheckResourceAttr("vercel_dns_record.mx", "ttl", "120"),
					resource.TestCheckResourceAttr("vercel_dns_record.mx", "mx_priority", "123"),
					resource.TestCheckResourceAttr("vercel_dns_record.mx", "value", "example.com."),
					testAccDNSRecordExists("vercel_dns_record.srv", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.srv", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.srv", "type", "SRV"),
					resource.TestCheckResourceAttr("vercel_dns_record.srv", "ttl", "120"),
					resource.TestCheckResourceAttr("vercel_dns_record.srv", "srv.port", "5000"),
					resource.TestCheckResourceAttr("vercel_dns_record.srv", "srv.weight", "120"),
					resource.TestCheckResourceAttr("vercel_dns_record.srv", "srv.priority", "27"),
					resource.TestCheckResourceAttr("vercel_dns_record.srv", "srv.target", "example.com."),
					testAccDNSRecordExists("vercel_dns_record.srv_no_target", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.srv_no_target", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.srv_no_target", "type", "SRV"),
					resource.TestCheckResourceAttr("vercel_dns_record.srv_no_target", "ttl", "120"),
					resource.TestCheckResourceAttr("vercel_dns_record.srv_no_target", "srv.port", "5000"),
					resource.TestCheckResourceAttr("vercel_dns_record.srv_no_target", "srv.weight", "120"),
					resource.TestCheckResourceAttr("vercel_dns_record.srv_no_target", "srv.priority", "27"),
					testAccDNSRecordExists("vercel_dns_record.txt", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.txt", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.txt", "type", "TXT"),
					resource.TestCheckResourceAttr("vercel_dns_record.txt", "ttl", "120"),
					resource.TestCheckResourceAttr("vercel_dns_record.txt", "value", "terraform testing"),
					testAccDNSRecordExists("vercel_dns_record.ns", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.ns", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.ns", "type", "NS"),
					resource.TestCheckResourceAttr("vercel_dns_record.ns", "ttl", "120"),
					resource.TestCheckResourceAttr("vercel_dns_record.ns", "value", "example.com."),
				),
			},
			{
				Config: testAccDNSRecordConfigUpdated(testDomain(), nameSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDNSRecordExists("vercel_dns_record.a_without_ttl", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.a", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.a", "type", "A"),
					resource.TestCheckResourceAttr("vercel_dns_record.a", "ttl", "120"),
					resource.TestCheckResourceAttr("vercel_dns_record.a", "value", "127.0.0.1"),
					testAccDNSRecordExists("vercel_dns_record.a", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.a", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.a", "type", "A"),
					resource.TestCheckResourceAttr("vercel_dns_record.a", "ttl", "60"),
					resource.TestCheckResourceAttr("vercel_dns_record.a", "value", "192.168.0.1"),
					testAccDNSRecordExists("vercel_dns_record.aaaa", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.aaaa", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.aaaa", "type", "AAAA"),
					resource.TestCheckResourceAttr("vercel_dns_record.aaaa", "ttl", "60"),
					resource.TestCheckResourceAttr("vercel_dns_record.aaaa", "value", "::0"),
					testAccDNSRecordExists("vercel_dns_record.alias", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.alias", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.alias", "type", "ALIAS"),
					resource.TestCheckResourceAttr("vercel_dns_record.alias", "ttl", "60"),
					resource.TestCheckResourceAttr("vercel_dns_record.alias", "value", "example2.com."),
					testAccDNSRecordExists("vercel_dns_record.caa", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.caa", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.caa", "type", "CAA"),
					resource.TestCheckResourceAttr("vercel_dns_record.caa", "ttl", "60"),
					resource.TestCheckResourceAttr("vercel_dns_record.caa", "value", "1 issue \"letsencrypt.org\""),
					testAccDNSRecordExists("vercel_dns_record.cname", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.cname", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.cname", "type", "CNAME"),
					resource.TestCheckResourceAttr("vercel_dns_record.cname", "ttl", "60"),
					resource.TestCheckResourceAttr("vercel_dns_record.cname", "value", "example2.com."),
					testAccDNSRecordExists("vercel_dns_record.mx", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.mx", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.mx", "type", "MX"),
					resource.TestCheckResourceAttr("vercel_dns_record.mx", "ttl", "60"),
					resource.TestCheckResourceAttr("vercel_dns_record.mx", "mx_priority", "333"),
					resource.TestCheckResourceAttr("vercel_dns_record.mx", "value", "example2.com."),
					testAccDNSRecordExists("vercel_dns_record.srv", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.srv", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.srv", "type", "SRV"),
					resource.TestCheckResourceAttr("vercel_dns_record.srv", "ttl", "60"),
					resource.TestCheckResourceAttr("vercel_dns_record.srv", "srv.port", "6000"),
					resource.TestCheckResourceAttr("vercel_dns_record.srv", "srv.weight", "60"),
					resource.TestCheckResourceAttr("vercel_dns_record.srv", "srv.priority", "127"),
					resource.TestCheckResourceAttr("vercel_dns_record.srv", "srv.target", "example2.com."),
					testAccDNSRecordExists("vercel_dns_record.txt", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.txt", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.txt", "type", "TXT"),
					resource.TestCheckResourceAttr("vercel_dns_record.txt", "ttl", "60"),
					resource.TestCheckResourceAttr("vercel_dns_record.txt", "value", "terraform testing two"),
					testAccDNSRecordExists("vercel_dns_record.ns", ""),
					resource.TestCheckResourceAttr("vercel_dns_record.ns", "domain", testDomain()),
					resource.TestCheckResourceAttr("vercel_dns_record.ns", "type", "NS"),
					resource.TestCheckResourceAttr("vercel_dns_record.ns", "ttl", "60"),
					resource.TestCheckResourceAttr("vercel_dns_record.ns", "value", "example2.com."),
				),
			},
			{
				ResourceName:      "vercel_dns_record.a",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "vercel_dns_record.aaaa",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "vercel_dns_record.alias",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "vercel_dns_record.caa",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "vercel_dns_record.cname",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "vercel_dns_record.mx",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "vercel_dns_record.srv",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "vercel_dns_record.txt",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccDNSRecordConfig(testDomain, nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_dns_record" "a_without_ttl" {
  domain = "%[1]s"
  name  = "test-acc-%[2]s-a-without-ttl-record"
  type  = "A"
  value = "127.0.0.1"
}
resource "vercel_dns_record" "a" {
  domain = "%[1]s"
  name  = "test-acc-%[2]s-a-record"
  type  = "A"
  ttl   = 120
  value = "127.0.0.1"
}
resource "vercel_dns_record" "aaaa" {
  domain = "%[1]s"
  name  = "test-acc-%[2]s-aaaa-record"
  type  = "AAAA"
  ttl   = 120
  value = "::1"
}
resource "vercel_dns_record" "alias" {
  domain = "%[1]s"
  name  = "test-acc-%[2]s-alias"
  type  = "ALIAS"
  ttl   = 120
  value = "example.com."
}
resource "vercel_dns_record" "caa" {
  domain = "%[1]s"
  name   = "test-acc-%[2]s-caa"
  type   = "CAA"
  ttl    = 120
  value  = "0 issue \"letsencrypt.org\""
}
resource "vercel_dns_record" "cname" {
  domain = "%[1]s"
  name  = "test-acc-%[2]s-cname"
  type  = "CNAME"
  ttl   = 120
  value = "example.com."
}
resource "vercel_dns_record" "mx" {
  domain = "%[1]s"
  name        = "test-acc-%[2]s-mx"
  type        = "MX"
  ttl         = 120
  mx_priority = 123
  value       = "example.com."
}
resource "vercel_dns_record" "srv" {
  domain = "%[1]s"
  name = "test-acc-%[2]s-srv"
  type = "SRV"
  ttl  = 120
  srv = {
      port     = 5000
      weight   = 120
      priority = 27
      target   = "example.com."
  }
}
resource "vercel_dns_record" "srv_no_target" {
  domain = "%[1]s"
  name = "test-acc-%[2]s-srv-no-target"
  type = "SRV"
  ttl  = 120
  srv = {
      port     = 5000
      weight   = 120
      priority = 27
      target = ""
  }
}
resource "vercel_dns_record" "txt" {
  domain = "%[1]s"
  name = "test-acc-%[2]s-txt"
  type = "TXT"
  ttl  = 120
  value = "terraform testing"
}
resource "vercel_dns_record" "ns" {
  domain = "%[1]s"
  name = "test-acc-%[2]s-ns"
  type = "NS"
  ttl  = 120
  value = "example.com."
}
`, testDomain, nameSuffix)
}

func testAccDNSRecordConfigUpdated(testDomain, nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_dns_record" "a_without_ttl" {
  domain = "%[1]s"
  name  = "test-acc-%[2]s-a-without-ttl-record"
  type  = "A"
  ttl   = 120
  value = "127.0.0.1"
}
resource "vercel_dns_record" "a" {
  domain = "%[1]s"
  name  = "test-acc-%[2]s-a-record-updated"
  type  = "A"
  ttl   = 60
  value = "192.168.0.1"
}
resource "vercel_dns_record" "aaaa" {
  domain = "%[1]s"
  name  = "test-acc-%[2]s-aaaa-record-updated"
  type  = "AAAA"
  ttl   = 60
  value = "::0"
}
resource "vercel_dns_record" "alias" {
  domain = "%[1]s"
  name  = "test-acc-%[2]s-alias-updated"
  type  = "ALIAS"
  ttl   = 60
  value = "example2.com."
}
resource "vercel_dns_record" "caa" {
  domain = "%[1]s"
  name   = "test-acc-%[2]s-caa-updated"
  type   = "CAA"
  ttl    = 60
  value  = "1 issue \"letsencrypt.org\""
}
resource "vercel_dns_record" "cname" {
  domain = "%[1]s"
  name  = "test-acc-%[2]s-cname-updated"
  type  = "CNAME"
  ttl   = 60
  value = "example2.com."
}
resource "vercel_dns_record" "mx" {
  domain = "%[1]s"
  name        = "test-acc-%[2]s-mx-updated"
  type        = "MX"
  ttl         = 60
  mx_priority = 333
  value       = "example2.com."
}
resource "vercel_dns_record" "srv" {
  domain = "%[1]s"
  name = "test-acc-%[2]s-srv-updated"
  type = "SRV"
  ttl  = 60
  srv = {
      port     = 6000
      weight   = 60
      priority = 127
      target   = "example2.com."
  }
}
resource "vercel_dns_record" "txt" {
  domain = "%[1]s"
  name = "test-acc-%[2]s-txt-updated"
  type = "TXT"
  ttl  = 60
  value = "terraform testing two"
}
resource "vercel_dns_record" "ns" {
  domain = "%[1]s"
  name = "test-acc-%[2]s-ns-updated"
  type = "NS"
  ttl  = 60
  value = "example2.com."
}
`, testDomain, nameSuffix)
}
