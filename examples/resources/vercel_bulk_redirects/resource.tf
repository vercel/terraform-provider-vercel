resource "vercel_project" "example" {
  name = "example-project"
}

resource "vercel_bulk_redirects" "example" {
  project_id = vercel_project.example.id

  redirects = [
    {
      source                = "/old-path"
      destination           = "/new-path"
      status_code           = 307
      case_sensitive        = false
      query                 = false
    },
    {
      source                = "/blog"
      destination           = "https://example.com/blog"
      status_code           = 308
      case_sensitive        = true
      query                 = true
    },
  ]
}
