# If importing into a personal account, or with a team configured on
# the provider, simply use the project_id and environment variable id.
# - project_id can be found in the project `settings` tab in the Vercel UI.
# - environment variable id is hard to find, but can be taken from the network tab, inside developer tools, on the project page.
#
# Note also, that the value field for sensitive environment variables will be imported as `null`.
terraform import vercel_project_environment_variable.example prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/FdT2e1E5Of6Cihmt

# Alternatively, you can import via the team_id, project_id and
# environment variable id.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - project_id can be found in the project `settings` tab in the Vercel UI.
# - environment variable id is hard to find, but can be taken from the network tab, inside developer tools, on the project page.
#
# Note also, that the value field for sensitive environment variables will be imported as `null`.
terraform import vercel_project_environment_variable.example team_xxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/FdT2e1E5Of6Cihmt
