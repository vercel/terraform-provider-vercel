resource "vercel_custom_certificate" "example" {
  private_key                       = file("private.key")
  certificate                       = file("certificate.crt")
  certificate_authority_certificate = file("ca.crt")
}
