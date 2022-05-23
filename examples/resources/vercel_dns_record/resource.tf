resource "vercel_dns_record" "a" {
  domain = "example.com"
  name   = "subdomain" # for subdomain.example.com
  type   = "A"
  ttl    = 60
  value  = "192.168.0.1"
}

resource "vercel_dns_record" "aaaa" {
  domain = "example.com"
  name   = "subdomain"
  type   = "AAAA"
  ttl    = 60
  value  = "::0"
}

resource "vercel_dns_record" "alias" {
  domain = "example.com"
  name   = "subdomain"
  type   = "ALIAS"
  ttl    = 60
  value  = "example2.com."
}

resource "vercel_dns_record" "caa" {
  domain = "example.com"
  name   = "subdomain"
  type   = "CAA"
  ttl    = 60
  value  = "1 issue \"letsencrypt.org\""
}

resource "vercel_dns_record" "cname" {
  domain = "example.com"
  name   = "subdomain"
  type   = "CNAME"
  ttl    = 60
  value  = "example2.com."
}

resource "vercel_dns_record" "mx" {
  domain      = "example.com"
  name        = "subdomain"
  type        = "MX"
  ttl         = 60
  mx_priority = 333
  value       = "example2.com."
}

resource "vercel_dns_record" "srv" {
  domain = "example.com"
  name   = "subdomain"
  type   = "SRV"
  ttl    = 60
  srv = {
    port     = 6000
    weight   = 60
    priority = 127
    target   = "example2.com."
  }
}

resource "vercel_dns_record" "txt" {
  domain = "example.com"
  name   = "subdomain"
  type   = "TXT"
  ttl    = 60
  value  = "some text value"
}
