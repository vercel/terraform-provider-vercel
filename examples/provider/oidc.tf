provider "vercel" {
  # Or omit this for the path to be read from the
  # VERCEL_TERRAFORM_OIDC_TOKEN_FILE_PATH environment variable
  oidc_token_file_path = "/var/run/secrets/tokens/vercel"

  # The team whose OIDC federation policy matches the token
  team = "your_team_slug_or_id"
}
