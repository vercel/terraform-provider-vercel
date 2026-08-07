resource "vercel_kms_issuer" "example" {
  name      = "my-issuer"
  algorithm = "RS512"
}

# Import an externally-generated key instead of having Vercel generate one.
resource "vercel_kms_issuer" "external" {
  name       = "my-external-issuer"
  import_key = file("${path.module}/private-key.pem")
  key_id     = "my-key-id"
}
