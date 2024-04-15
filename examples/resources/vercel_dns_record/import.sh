# If importing into a personal account, or with a team configured on
# the provider, simply use the record id.
# - record_id can be taken from the network tab inside developer tools, while you are on the domains page,
# or can be queried from the Vercel API directly (https://vercel.com/docs/rest-api/endpoints/dns#list-existing-dns-records).
terraform import vercel_dns_record.example rec_xxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Alternatively, you can import via the team_id and record_id.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - record_id can be taken from the network tab inside developer tools, while you are on the domains page,
# or can be queried from the Vercel API directly (https://vercel.com/docs/rest-api/endpoints/dns#list-existing-dns-records).
terraform import vercel_dns_record.example team_xxxxxxxxxxxxxxxxxxxxxxxx/rec_xxxxxxxxxxxxxxxxxxxxxxxxxxxx

