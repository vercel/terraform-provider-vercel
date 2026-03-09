# If importing into a personal account, or with a team configured on
# the provider, use the Blob store id and connection id.
terraform import vercel_blob_project_connection.example store_xxxxxxxxxxxxxxxxxx/spc_xxxxxxxxxxxxxxxxxx

# Alternatively, you can import via the team_id, Blob store id, and connection id.
terraform import vercel_blob_project_connection.example team_xxxxxxxxxxxxxxxxxxxxxxxx/store_xxxxxxxxxxxxxxxxxx/spc_xxxxxxxxxxxxxxxxxx
