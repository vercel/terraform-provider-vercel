resource "vercel_blob_store" "example" {
  name = "example-blob-store"
}

data "vercel_blob_store" "example" {
  id = vercel_blob_store.example.id
}
