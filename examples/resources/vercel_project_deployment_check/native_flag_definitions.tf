resource "vercel_project_deployment_check" "native_flag_definitions" {
  project_id = vercel_project.example.id
  name       = "Flag Definitions Present"
  requires   = "build-ready"
  blocks     = "deployment-alias"
  targets    = ["preview", "production"]

  source = {
    kind     = "vercel"
    sub_kind = "vercel-native-check"
    job_name = "flag-definitions-present"
  }
}
