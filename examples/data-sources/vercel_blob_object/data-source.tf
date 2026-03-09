resource "vercel_blob_store" "example" {
  name = "example-blob-store"
}

resource "vercel_blob_object" "example" {
  store_id              = vercel_blob_store.example.id
  pathname              = "branding/logo.svg"
  source                = "${path.module}/files/logo.svg"
  content_type          = "image/svg+xml"
  cache_control_max_age = 86400
}

data "vercel_blob_object" "example" {
  store_id = vercel_blob_store.example.id
  pathname = vercel_blob_object.example.pathname
}
