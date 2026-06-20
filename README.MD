# SQL Terraform Provider

The SQL Terraform Provider exposes a single `sql` resource that runs an `up` SQL script when the resource is created and a `down` SQL script when it is destroyed. It is meant for managing database-level entities — databases, roles, schemas, grants — alongside the rest of your infrastructure, not as a general-purpose schema migration tool.

PostgreSQL only. The driver is hardcoded to `postgres` (`lib/pq`); this is by design, not a temporary limitation.

```hcl
resource "sql" "task" {
    database = "app_db"
    up       = "CREATE ROLE owner WITH LOGIN VALID UNTIL 'infinity'"
    down     = "DROP ROLE owner"
}
```


## Status

Stable and in production use. Read the Resource Behavior section before you wire it into anything — it doesn't reconcile drift, and `up` is immutable once created. Know what you're signing up for and it works fine.


## Features
- Runs `up` SQL on create and `down` SQL on destroy (the `down` script is optional).
- Control execution order with Terraform's `depends_on`. Without it, resources apply in parallel.
- No long refresh/reconcile pass — the provider does not treat database objects as managed state.
- No persistent connection. A connection is opened only when an `up`/`down` script actually runs.
- Handles multi-statement scripts, including statements that cannot run inside a transaction (e.g. `CREATE DATABASE`).


## Provider Configuration

Define the provider in a `provider.tf`:

```hcl
terraform {
  required_version = ">= 1.0"

  required_providers {
    sql = {
      source  = "pilat/sql"
      version = "0.0.8" # use the latest published version
    }
  }
}

provider "sql" {
    dsn = "postgresql://admin:pass@postgres_host?sslmode=disable"
}
```

- `dsn` (string, **sensitive**) — connection string. Defaults to the `SQL_DSN` environment variable. Two forms are accepted:
  - URL form: `postgresql://user:pass@host/db?sslmode=disable`
  - keyword form: `user=foo password=bar host=localhost dbname=baz sslmode=disable`

  The database part of the DSN is rewritten for every operation using the resource's `database` attribute, so whatever database the DSN points at only acts as a default.
- `timeout` (int, seconds) — connection max lifetime. Defaults to the `SQL_TIMEOUT` environment variable, or `600` if unset.


## Usage

```hcl
resource "sql" "migration_1" {
    database = "postgres"
    up       = "CREATE ROLE test_role WITH LOGIN VALID UNTIL 'infinity'"
    down     = "DROP ROLE test_role"
}

resource "sql" "migration_2" {
    depends_on = [ sql.migration_1 ]
    database   = "postgres"
    up = <<-EOF
        GRANT test_role TO CURRENT_USER;
        CREATE DATABASE test_db OWNER test_role;
        REVOKE test_role FROM CURRENT_USER;
    EOF
    down = "DROP DATABASE test_db"
}

resource "sql" "migration_3" {
    depends_on = [ sql.migration_2 ]
    database   = "test_db"
    up = <<-EOF
        GRANT ALL ON DATABASE test_db TO test_role;
        ALTER SCHEMA public OWNER TO test_role;
    EOF
}
```

The `depends_on` directive enforces execution order. Without it, blocks execute in parallel.


## Resource Behavior

This resource does not behave like a typical "managed object" provider. Read this before you rely on it.

- **`up` is immutable.** Once the resource exists, changing `up` is rejected at plan time:

  ```
  changing the `up` attribute is not allowed after the resource has been created
  ```

  Changing the script means creating a new resource.
- **`up`/`down` run statement by statement.** Multi-statement scripts are split on a semicolon followed by a newline (`;\n` or `;\r\n`) and each statement is executed separately, with **no wrapping transaction**. That is deliberate — it lets statements such as `CREATE DATABASE`, which PostgreSQL refuses to run inside a transaction block, succeed.
- **Read is a no-op.** The provider never reads the database back, so it does not detect drift or out-of-band changes. If someone drops your role by hand, Terraform won't notice.
- **Update runs no SQL.** You may edit `database` or `down` on an existing resource; that updates Terraform state and emits a warning, but it does **not** touch the database. The purpose is to let you correct a wrong `down`/`database` before destroying, so the right `down` runs at destroy time.
- **Resource ID** is the first 8 hex characters of `sha256(up)`.


## Importing

Import takes a JSON object as the id. The `database`, `up`, and `down` values must match your configuration; the real id is recomputed as `sha256(up)[:8]` during import.

```sh
terraform import sql.example '{"database":"postgres","up":"CREATE ROLE r WITH LOGIN","down":"DROP ROLE r"}'
```


## Development / Testing

Two Makefile targets:

- `make test` — fast unit suite using sqlmock. No database, no Docker.
- `make test-e2e` — acceptance tests against a throwaway Docker PostgreSQL. Requires `terraform` **or** `tofu` on `PATH` and a running Docker daemon. The container is torn down automatically afterwards.

The acceptance tests are gated by `TF_ACC`. A plain `go test ./...` (what `make test` runs) skips them and exercises only the unit suite.


## Disclaimer

This provider runs whatever SQL you hand it. There's no sandbox and no migration framework underneath — your `up`/`down` scripts are executed as-is. Read the SQL before you apply, and keep in mind that a careless `terraform apply` can do real damage when the statements touch roles, grants, or whole databases.

It's also not a general-purpose migration tool. It's for managing database-level objects (databases, roles, schemas, grants) next to the rest of your infra.


## Contribute

Open source, contributions welcome. Bug reports, feature ideas, pull requests — all of it.


## Licensing

The SQL Terraform Provider is licensed under the MIT License. Feel free to use, modify, and distribute the code under the terms of the license.
