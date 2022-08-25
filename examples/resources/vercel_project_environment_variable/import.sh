# Import via the team_id, project_id and environment variable id.
# team_id can be found in the team `settings` tab in the Vercel UI.
# environment variable id can be taken from the network tab on the project page.
terraform import vercel_project_environment_variable.example team_xxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/FdT2e1E5Of6Cihmt

# If importing without a team, simply use the project_id and environment variable id.
terraform import vercel_project_environment_variable.example_git_branch prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/FdT2e1E5Of6Cihmt
