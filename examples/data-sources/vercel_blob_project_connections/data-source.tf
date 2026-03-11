resource "vercel_blob_store" "example" {
  name = "example-blob-store"
}

resource "vercel_project" "example" {
  name = "example-blob-project"
}

resource "vercel_blob_project_connection" "example" {
  blob_store_id = vercel_blob_store.example.id
  project_id    = vercel_project.example.id
}

data "vercel_blob_project_connections" "example" {
  store_id = vercel_blob_store.example.id
}
