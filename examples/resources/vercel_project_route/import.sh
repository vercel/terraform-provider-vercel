# If importing with a team configured on the provider, use the project ID and route ID.
# - project_id can be found in the project `settings` tab in the Vercel UI.
# - route_id can be read from `data.vercel_project_routes` or the Vercel routing-rules API.
terraform import vercel_project_route.example prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/rt_xxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Alternatively, you can import via the team_id, project_id, and route_id.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - project_id can be found in the project `settings` tab in the Vercel UI.
# - route_id can be read from `data.vercel_project_routes` or the Vercel routing-rules API.
terraform import vercel_project_route.example team_xxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/rt_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
