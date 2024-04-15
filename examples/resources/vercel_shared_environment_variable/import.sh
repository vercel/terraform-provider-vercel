# You can import via the team_id and environment variable id.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - environment variable id can be taken from the network tab inside developer tools, while you are on the project page.
#
# Note also, that the value field for sensitive environment variables will be imported as `null`.
terraform import vercel_shared_environment_variable.example team_xxxxxxxxxxxxxxxxxxxxxxxx/env_yyyyyyyyyyyyy
