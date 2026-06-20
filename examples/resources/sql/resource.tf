resource "sql" "role" {
  database = "postgres"
  up       = "CREATE ROLE test_role WITH LOGIN VALID UNTIL 'infinity'"
  down     = "DROP ROLE test_role"
}

# depends_on enforces execution order. Without it, resources apply in parallel.
resource "sql" "database" {
  depends_on = [sql.role]

  database = "postgres"
  up       = <<-EOF
    GRANT test_role TO CURRENT_USER;
    CREATE DATABASE test_db OWNER test_role;
    REVOKE test_role FROM CURRENT_USER;
  EOF
  down     = "DROP DATABASE test_db"
}
