resource "vercel_domain" "example" {
  name = "example.com"
  # Create a DNS zone on Vercel and use Vercel's nameservers.
  zone = true
}
