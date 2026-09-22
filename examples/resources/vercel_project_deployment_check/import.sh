# With an explicit team:
terraform import vercel_project_deployment_check.e2e team_xxx/prj_xxx/check_xxx

# With the team configured on the provider:
terraform import vercel_project_deployment_check.e2e prj_xxx/check_xxx

# Project names are also supported. Use the same ID or name as project_id in config.
terraform import vercel_project_deployment_check.e2e team_xxx/my-project/check_xxx
