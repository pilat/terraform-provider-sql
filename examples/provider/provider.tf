terraform {
  required_providers {
    sql = {
      source = "pilat/sql"
    }
  }
}

provider "sql" {
  # Connection string for the target PostgreSQL. Falls back to the SQL_DSN env var.
  # URL form:     postgresql://user:pass@host/db?sslmode=disable
  # keyword form: user=foo password=bar host=localhost dbname=baz sslmode=disable
  #
  # The database part is rewritten per-operation from each resource's `database`
  # attribute, so this only acts as a default.
  dsn = "postgresql://admin:pass@postgres_host?sslmode=disable"
}
