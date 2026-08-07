resource "vercel_kms_issuer" "example" {
  name = "my-issuer"
}

resource "vercel_kms_signing_key" "example" {
  issuer_id = vercel_kms_issuer.example.id

  # Change any value in keepers to rotate in a new signing key.
  keepers = {
    rotated_at = "2024-01-01"
  }
}
