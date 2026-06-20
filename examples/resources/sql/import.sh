# Import takes a JSON object as the id. The database/up/down values must match
# your configuration; the real id is recomputed as sha256(up)[:8] on import.
terraform import sql.example '{"database":"postgres","up":"CREATE ROLE r WITH LOGIN","down":"DROP ROLE r"}'
