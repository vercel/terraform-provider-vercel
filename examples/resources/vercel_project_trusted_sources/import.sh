# Import the existing Trusted Sources configuration, without importing the project itself.
terraform import vercel_project_trusted_sources.existing team_example/prj_existing

# With a provider team configured, the project ID alone is sufficient.
terraform import vercel_project_trusted_sources.existing prj_existing
