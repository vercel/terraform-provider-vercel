resource "vercel_project_deployment_check" "native_mfe_config" {
  project_id = vercel_project.example.id
  name       = "Microfrontends Config Present"
  requires   = "build-ready"
  blocks     = "deployment-alias"
  targets    = ["production"]

  source = {
    kind     = "vercel"
    sub_kind = "vercel-native-check"
    job_name = "mfe-config-present"
    origin   = "platform"
  }
}
