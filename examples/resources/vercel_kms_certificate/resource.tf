resource "vercel_kms_issuer" "example" {
  name = "my-issuer"
}

resource "vercel_kms_certificate" "example" {
  issuer_id = vercel_kms_issuer.example.id

  subject = {
    ou = "Engineering"
    c  = "US"
  }

  # Change any value in keepers to mint a fresh certificate.
  keepers = {
    minted_at = "2024-01-01"
  }
}
