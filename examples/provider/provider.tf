# Configure the Vercel provider using the required_providers stanza
# required with Terraform 0.13 and beyond. You may optionally use a
# version directive to prevent breaking changes occurring unannounced.
terraform {
  required_providers {
    cloudflare = {
      source  = "vercel/vercel"
      version = "~> 0.1"
    }
  }
}

provider "vercel" {
  # Or omit this for the api_token to be read
  # from the VERCEL_API_TOKEN environment variable
  api_token = var.vercel_api_token
}
