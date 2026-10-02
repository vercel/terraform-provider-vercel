data "vercel_shareable_link" "preview" {
  alias = "my-project-git-staging.vercel.app"
}

output "preview_share_url" {
  value     = data.vercel_shareable_link.preview.url
  sensitive = true
}
