terraform {
  required_providers {
    vercel = {
      source = "vercel/vercel"
    }
  }
}

resource "vercel_project_rolling_release" "example" {
  project_id = "prj_9lRsbRoK8DCtxa4CmUu5rWfSaS86"
  team_id    = "team_4FWx5KQoszRi0ZmM9q9IBoKG"
  rolling_release = {
    enabled         = true
    advancement_type = "automatic"
    stages          = [
      {
        duration = 1
        require_approval  = false
        target_percentage = 5  # Start with 5%
      },
      {
        duration = 1
        require_approval  = false
        target_percentage = 50  # Then 50%
      },
      {
        require_approval  = false  # No duration for last stage
        target_percentage = 100  # Finally 100%
      }
    ]
  }
} 