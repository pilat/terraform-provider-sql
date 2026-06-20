package e2e

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	_ "github.com/lib/pq"

	provider "github.com/pilat/terraform-provider-sql/sql"
)

// These tests run the in-process provider against a real Postgres and verify
// side effects by querying the database directly: Read is an intentional no-op,
// so Terraform state reflects nothing about what actually happened in the DB.
// Gated by TF_ACC; plain `go test ./...` skips every TestAcc*.

const (
	roleExistsQuery = "SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=$1)"
	dbExistsQuery   = "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=$1)"

	// lib/pq folds unquoted identifiers to lower case, so names must be all
	// lower case for the pg_roles/pg_database lookups to match.
	identCharset = "abcdefghijklmnopqrstuvwxyz"
)

var providerFactories = map[string]func() (*schema.Provider, error){
	"sql": func() (*schema.Provider, error) { return provider.Provider(), nil },
}

func TestAccSQL_createRunsUpDestroyRunsDown(t *testing.T) {
	role := randName("role")
	config := fmt.Sprintf(`
resource "sql" "a" {
  database = "postgres"
  up       = "CREATE ROLE %[1]s WITH LOGIN"
  down     = "DROP ROLE %[1]s"
}
`, role)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      checkRoleAbsent(t, role),
		Steps: []resource.TestStep{{
			Config: config,
			Check:  checkRoleExists(t, role),
		}},
	})
}

func TestAccSQL_multiStatementSplit(t *testing.T) {
	role := randName("role")
	db := randName("db")
	// The \n escapes become real newlines after HCL parsing; the provider splits
	// on ";\n", so a passing checkDatabaseExists proves all three statements ran.
	config := fmt.Sprintf(`
resource "sql" "role" {
  database = "postgres"
  up       = "CREATE ROLE %[1]s WITH LOGIN"
  down     = "DROP ROLE %[1]s"
}

resource "sql" "db" {
  depends_on = [sql.role]
  database   = "postgres"
  up         = "GRANT %[1]s TO CURRENT_USER;\nCREATE DATABASE %[2]s OWNER %[1]s;\nREVOKE %[1]s FROM CURRENT_USER;"
  down       = "DROP DATABASE %[2]s"
}
`, role, db)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			checkDatabaseAbsent(t, db),
			checkRoleAbsent(t, role),
		),
		Steps: []resource.TestStep{{
			Config: config,
			Check:  checkDatabaseExists(t, db),
		}},
	})
}

func TestAccSQL_multiDatabaseTargeting(t *testing.T) {
	role := randName("role")
	db := randName("db")
	table := randName("tbl")
	// sql.obj targets the freshly created database; a table found there proves the
	// provider rewrote the DSN path per the resource's `database` attribute.
	config := fmt.Sprintf(`
resource "sql" "role" {
  database = "postgres"
  up       = "CREATE ROLE %[1]s WITH LOGIN"
  down     = "DROP ROLE %[1]s"
}

resource "sql" "db" {
  depends_on = [sql.role]
  database   = "postgres"
  up         = "GRANT %[1]s TO CURRENT_USER;\nCREATE DATABASE %[2]s OWNER %[1]s;\nREVOKE %[1]s FROM CURRENT_USER;"
  down       = "DROP DATABASE %[2]s"
}

resource "sql" "obj" {
  depends_on = [sql.db]
  database   = "%[2]s"
  up         = "CREATE TABLE %[3]s (id int)"
  down       = "DROP TABLE %[3]s"
}
`, role, db, table)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			checkDatabaseAbsent(t, db),
			checkRoleAbsent(t, role),
		),
		Steps: []resource.TestStep{{
			Config: config,
			Check:  checkTableExists(t, db, table),
		}},
	})
}

func TestAccSQL_upIsImmutable(t *testing.T) {
	role := randName("role")
	base := `
resource "sql" "a" {
  database = "postgres"
  up       = "%s"
  down     = "DROP ROLE %s"
}
`
	step1 := fmt.Sprintf(base, "CREATE ROLE "+role+" WITH LOGIN", role)
	step2 := fmt.Sprintf(base, "CREATE ROLE "+role+" WITH LOGIN VALID UNTIL 'infinity'", role)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      checkRoleAbsent(t, role),
		Steps: []resource.TestStep{
			{Config: step1, Check: checkRoleExists(t, role)},
			{Config: step2, ExpectError: regexp.MustCompile(regexp.QuoteMeta("changing the `up` attribute is not allowed after the resource has been created"))},
		},
	})
}

func TestAccSQL_downChangeIsNoOp(t *testing.T) {
	role := randName("role")
	base := `
resource "sql" "a" {
  database = "postgres"
  up       = "CREATE ROLE %[1]s WITH LOGIN"
  down     = "%[2]s"
}
`
	step1 := fmt.Sprintf(base, role, "DROP ROLE "+role)
	step2 := fmt.Sprintf(base, role, "DROP ROLE IF EXISTS "+role)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      checkRoleAbsent(t, role),
		Steps: []resource.TestStep{
			{Config: step1, Check: checkRoleExists(t, role)},
			// Update runs no SQL (warns only): the apply succeeds and the role is
			// untouched. The framework does not expose warning text, so unchanged
			// DB state is the strongest assertion available here.
			{Config: step2, Check: checkRoleExists(t, role)},
		},
	})
}

func TestAccSQL_import(t *testing.T) {
	role := randName("role")
	config := fmt.Sprintf(`
resource "sql" "a" {
  database = "postgres"
  up       = "CREATE ROLE %[1]s WITH LOGIN"
  down     = "DROP ROLE %[1]s"
}
`, role)
	// up/down must match the config byte-for-byte so the recomputed sha256[:8] id
	// and the imported attributes line up under ImportStateVerify.
	importID := fmt.Sprintf(`{"database":"postgres","up":"CREATE ROLE %[1]s WITH LOGIN","down":"DROP ROLE %[1]s"}`, role)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      checkRoleAbsent(t, role),
		Steps: []resource.TestStep{
			{Config: config, Check: checkRoleExists(t, role)},
			{
				ResourceName:      "sql.a",
				ImportState:       true,
				ImportStateId:     importID,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSQL_noDrift(t *testing.T) {
	role := randName("role")
	config := fmt.Sprintf(`
resource "sql" "a" {
  database = "postgres"
  up       = "CREATE ROLE %[1]s WITH LOGIN"
  down     = "DROP ROLE %[1]s"
}
`, role)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      checkRoleAbsent(t, role),
		Steps: []resource.TestStep{
			{Config: config, Check: checkRoleExists(t, role)},
			// A PlanOnly step fails on a non-empty plan: guards against the no-op
			// Read ever starting to produce drift.
			{Config: config, PlanOnly: true},
		},
	})
}

func testAccPreCheck(t *testing.T) {
	// Unset SQL_DSN is a deliberate opt-out (skip). But once it IS set, an
	// unreachable database is a hard failure, not a skip — otherwise a broken
	// Postgres in CI would let every TestAcc* skip and the job would go green
	// having verified nothing.
	dsn := os.Getenv("SQL_DSN")
	if dsn == "" {
		t.Skip("SQL_DSN not set; skipping acceptance test")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open SQL_DSN: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Postgres unreachable at SQL_DSN: %v", err)
	}
}

// dsnFor rewrites SQL_DSN to point at dbName, mirroring the provider's own DSN
// path rewrite. It assumes the URL form (postgresql://...), which is what the
// Makefile and CI always set — the keyword form is not needed here.
func dsnFor(t *testing.T, dbName string) string {
	t.Helper()

	u, err := url.Parse(os.Getenv("SQL_DSN"))
	if err != nil {
		t.Fatalf("parse SQL_DSN: %v", err)
	}
	u.Path = "/" + dbName

	return u.String()
}

// pgConn opens a verification connection against dbName, derived from SQL_DSN.
// MaxIdleConns(0) keeps no connection pooled into the target database, otherwise
// a later DROP DATABASE fails with "database is being accessed by other users".
func pgConn(t *testing.T, dbName string) *sql.DB {
	t.Helper()

	db, err := sql.Open("postgres", dsnFor(t, dbName))
	if err != nil {
		t.Fatalf("open verification connection to %q: %v", dbName, err)
	}
	db.SetMaxIdleConns(0)
	t.Cleanup(func() { db.Close() })

	return db
}

func checkRoleExists(t *testing.T, name string) resource.TestCheckFunc {
	return checkPresence(t, "postgres", roleExistsQuery, name, "role", true)
}

func checkRoleAbsent(t *testing.T, name string) resource.TestCheckFunc {
	return checkPresence(t, "postgres", roleExistsQuery, name, "role", false)
}

func checkDatabaseExists(t *testing.T, name string) resource.TestCheckFunc {
	return checkPresence(t, "postgres", dbExistsQuery, name, "database", true)
}

func checkDatabaseAbsent(t *testing.T, name string) resource.TestCheckFunc {
	return checkPresence(t, "postgres", dbExistsQuery, name, "database", false)
}

func checkTableExists(t *testing.T, dbName, table string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		db := pgConn(t, dbName)

		var reg sql.NullString
		if err := db.QueryRow("SELECT to_regclass($1)", table).Scan(&reg); err != nil {
			return fmt.Errorf("to_regclass(%q) in %q: %w", table, dbName, err)
		}
		if !reg.Valid {
			return fmt.Errorf("expected table %q to exist in database %q", table, dbName)
		}

		return nil
	}
}

func checkPresence(t *testing.T, dbName, query, arg, kind string, want bool) resource.TestCheckFunc {
	return func(*terraform.State) error {
		db := pgConn(t, dbName)

		var got bool
		if err := db.QueryRow(query, arg).Scan(&got); err != nil {
			return fmt.Errorf("checking %s %q: %w", kind, arg, err)
		}
		if got != want {
			return fmt.Errorf("%s %q: present=%v, want %v", kind, arg, got, want)
		}

		return nil
	}
}

func randName(kind string) string {
	return fmt.Sprintf("tfacc_%s_%s", kind, acctest.RandStringFromCharSet(8, identCharset))
}
