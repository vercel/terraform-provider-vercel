resource "vercel_blob_store" "example" {
  name = "example-blob-store"
}

data "vercel_blob_store_secrets" "example" {
  store_id = vercel_blob_store.example.id
}
