# SQL Terraform Provider
Just a simple terraform provider to execute SQL statements.


## Status
🚧 Work in process. It is not ready for production use.


## How To Use

Define a `provider.tf`:

```hcl
terraform {
  required_version = ">= 1.0"

  required_providers {
    sql = {
      source = "pilat/sql"
      version = "0.0.4"
    }
  }
}

provider "sql" {
    dsn = "postgresql://admin:pass@host"
}
```

Use it in your terraform files:
```hcl
resource "sql" "migration_1" {
    database = "postgres"
    up = "CREATE ROLE test_role WITH LOGIN VALID UNTIL 'infinity';"
    down = "DROP ROLE test_role;"
}

resource "sql" "migration_2" {
    depends_on = [ sql.migration_1 ]
    database = "postgres"
    up = <<-EOF
        GRANT test_role TO admin;
        CREATE DATABASE test_db OWNER test_role;
        REVOKE test_role FROM admin;
    EOF
    down = "DROP DATABASE test_db;"
}

resource "sql" "migration_3" {
    depends_on = [ sql.migration_2 ]
    database = "test_db"
    up = <<-EOF
        GRANT ALL ON DATABASE test_db TO test_role;
        ALTER SCHEMA public OWNER TO test_role;
    EOF
}
```

You have to set `depends_on` to ensure the correct order of execution.