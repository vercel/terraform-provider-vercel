# Import via the team_id and record ID.
# Record ID can be taken from the network tab on the domains page.
terraform import vercel_dns_record.example team_xxxxxxxxxxxxxxxxxxxxxxxx/rec_xxxxxxxxxxxxxxxxxxxxxxxxxxxx

# If importing without a team, simply use the record ID.
terraform import vercel_dns_record.personal_example rec_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
