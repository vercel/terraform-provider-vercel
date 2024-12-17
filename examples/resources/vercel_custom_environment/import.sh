# If importing into a personal account, or with a team configured on
# the provider, simply use the project_id and custom environment name.
# - project_id can be found in the project `settings` tab in the Vercel UI.
terraform import vercel_custom_environment.example prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/example-custom-env

# Alternatively, you can import via the team_id, project_id and environment variable id.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - project_id can be found in the project `settings` tab in the Vercel UI.
#
# Note also, that the value field for sensitive environment variables will be imported as `null`.
terraform import vercel_custom_environment.example team_xxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/example-custom-env
