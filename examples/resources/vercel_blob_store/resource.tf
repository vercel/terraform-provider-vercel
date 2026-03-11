resource "vercel_blob_store" "example" {
  name   = "example-blob-store"
  access = "private"
  region = "iad1"
}
