# If importing into a personal account, or with a team configured on
# the provider, simply use the Blob store id.
terraform import vercel_blob_store.example store_xxxxxxxxxxxxxxxxxx

# Alternatively, you can import via the team_id and Blob store id.
terraform import vercel_blob_store.example team_xxxxxxxxxxxxxxxxxxxxxxxx/store_xxxxxxxxxxxxxxxxxx
