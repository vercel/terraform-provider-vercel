resource "vercel_custom_certificate" "example" {
  certificate_authority = "letsencrypt"
  key                   = file("private.pem")
  certificate           = file("certificate.crt")
}
