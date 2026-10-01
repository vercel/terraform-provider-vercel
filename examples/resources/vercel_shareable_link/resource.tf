resource "vercel_shareable_link" "preview" {
  alias       = "my-project-git-staging.vercel.app"
  ttl_seconds = 604800
  rotation_id = "1"
}

output "preview_share_url" {
  value     = vercel_shareable_link.preview.url
  sensitive = true
}
