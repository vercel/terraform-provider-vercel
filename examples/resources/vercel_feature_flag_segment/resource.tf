resource "vercel_project" "example" {
  name = "feature-flag-segment-example"
}

resource "vercel_feature_flag_segment" "example" {
  project_id  = vercel_project.example.id
  slug        = "internal-users"
  name        = "Internal Users"
  description = "Employees who should always see internal-only flag treatments"
  hint        = "user-email"
  include = [
    {
      entity    = "user"
      attribute = "email"
      values = [
        "alice@example.com",
        "bob@example.com",
      ]
    },
  ]

  exclude = [
    {
      entity    = "user"
      attribute = "id"
      values = [
        "contractor-123",
      ]
    },
  ]
}
