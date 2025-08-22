resource "vercel_project" "my_awesome_project" {
  name = "my-awesome-project"
}

# 
# "vercel_domain_config" Usage
# 
data "vercel_domain_config" "example_com" {
  domain             = "example.com"
  project_id_or_name = vercel_project.my_awesome_project.id
}

data "vercel_domain_config" "www_example_com" {
  domain             = "www.example.com"
  project_id_or_name = vercel_project.my_awesome_project.id
}

#
# External DNS provider example
# 
resource "aws_route53_record" "example_com_a" {
  zone_id = "...zone_id_from_somewhere..."
  name    = data.vercel_domain_config.example_com.domain
  type    = "A"
  ttl     = 300
  records = data.vercel_domain_config.example_com.recommended_ipv4s
}

resource "aws_route53_record" "www_example_com_cname" {
  zone_id = "...zone_id_from_somewhere..."
  name    = data.vercel_domain_config.www_example_com.domain
  type    = "CNAME"
  ttl     = 300
  records = [data.vercel_domain_config.www_example_com.recommended_cname]
}
