# Configure the Vercel provider using the required_providers stanza.
# You may optionally use a version directive to prevent breaking
# changes occurring unannounced.
terraform {
  required_providers {
    vercel = {
      source  = "vercel/vercel"
      version = "~> 1.0"
    }
  }
}

provider "vercel" {
  # Or omit this for the api_token to be read
  # from the VERCEL_API_TOKEN environment variable
  api_token = var.vercel_api_token

  # Optional default team for all resources
  team = "your_team_slug_or_id"
}
