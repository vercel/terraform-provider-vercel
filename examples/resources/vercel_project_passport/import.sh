# Adopt Passport settings on an existing project without importing the project itself.
terraform import vercel_project_passport.existing team_example/prj_existing

# With a provider team configured, the project ID alone is sufficient.
terraform import vercel_project_passport.existing prj_existing
