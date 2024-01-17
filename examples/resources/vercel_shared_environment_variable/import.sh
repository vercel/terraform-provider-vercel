# You can import via the team_id and environment variable id.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - environment variable id is hard to find, but can be taken from the network tab, inside developer tools, on the shared environment variable page.
terraform import vercel_shared_environment_variable.example team_xxxxxxxxxxxxxxxxxxxxxxxx/env_yyyyyyyyyyyyy
