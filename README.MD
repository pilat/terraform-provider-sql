# SQL Terraform Provider
The SQL Terraform Provider allows you to manage SQL operations within Terraform. This integration simplifies infrastructure management by enabling automated and version-controlled database tasks directly from your Terraform scripts.

```hcl
resource "sql" "task" {
    database = "app_db"
    up = "CREATE TABLE users (id SERIAL PRIMARY KEY, name text NOT NULL)"
    down = "DROP TABLE users"
}
```


## Status
This project is currently under development. While it is functional and can be used for testing and development, it is not yet recommended for production environments. 🚧


## Features
- Execute SQL scripts during Terraform's lifecycle stages: create ("up") and destroy ("down").
- Control the order of SQL execution using Terraform's `depends_on` directive.
- Avoid long refresh processes which usually occur when using database entities as managed objects in Terraform.
- Minimize the need for a persistent database connection. If no SQL statements have changed, no connection is required.
- Handle corner cases involving complex SQL statements.


## How To Use

To start using the SQL Terraform Provider, you need to define a `provider.tf` file:
```hcl
terraform {
  required_version = ">= 1.0"

  required_providers {
    sql = {
      source = "pilat/sql"
      version = "0.0.6"
    }
  }
}

provider "sql" {
    dsn = "postgresql://admin:pass@postgres_host"
}
```

Then, use the provider in your Terraform files:
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
        GRANT test_role TO CURRENT_USER;
        CREATE DATABASE test_db OWNER test_role;
        REVOKE test_role FROM CURRENT_USER;
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

The `depends_on` directive ensures the correct order of execution. Without `depends_on`, blocks execute in parallel.


## Working Principle
This provider operates by detecting and running the "up" SQL script when creating resources and the "down" SQL script when destroying resources (if provided).

If you update an existing block, it doesn't affect the data at the time of the update. However, if you modify the "down" script and subsequently remove your block, then the modified "down" script will be executed.


## Limitations
- Since this provider focuses on migration blocks rather than the database state, any changes to existing blocks won't affect the database directly.
- Multiline SQL statements must be delimited with ";\n" or ";\r\n".
- Only PostgreSQL is supported at the moment. MySQL support is planned for the future.
- Terraform documentation is not yet available.


## Contribute
As an open-source project, we welcome contributions! Whether you find a bug, have a great feature idea, or want to help with code, your involvement is warmly welcomed.


## Disclaimer
While the SQL Terraform Provider can execute SQL statements, it is crucial to note that it is not intended to serve as a comprehensive database migration tool. Its primary use case is managing database resources within your infrastructure, such as creating databases, roles, and similar entities.

This tool should be used wisely and with caution. It's always recommended to test your scripts in a development environment before running them in production. While the SQL Terraform Provider makes it easy to manage database operations alongside your cloud resources, it is essential to remember that Terraform is a powerful tool and changes made using Terraform can have significant impacts if not used carefully.


## Licensing
The SQL Terraform Provider is licensed under the MIT License. Feel free to use, modify, and distribute the code as per the terms of the license.
