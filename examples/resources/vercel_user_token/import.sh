# If importing into a personal account, or with a team configured on
# the provider, simply use the token ID.
# - token_id can be found in the Vercel UI under Account Settings, Tokens.
terraform import vercel_user_token.example 5d9f2ebd38ddca62e5d51e9c1704c72530bdc8bfdd41e782a6687c48399e8391

# Alternatively, you can import via the team_id and token_id.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - token_id can be found in the Vercel UI under Account Settings, Tokens.
terraform import vercel_user_token.example team_xxxxxxxxxxxxxxxxxxxxxxxx/5d9f2ebd38ddca62e5d51e9c1704c72530bdc8bfdd41e782a6687c48399e8391
