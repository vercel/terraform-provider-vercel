# Import all currently enabled methods, without importing the project itself.
# Include every imported method you want to retain in your configuration.
terraform import vercel_project_deployment_protection.existing team_example/prj_existing

# With a provider team configured, the project ID alone is sufficient.
terraform import vercel_project_deployment_protection.existing prj_existing
