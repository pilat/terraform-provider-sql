package sql

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetDSN(t *testing.T) {
	tests := []struct {
		dsn    string
		db     string
		expect string
	}{
		{"postgresql://user:pass@localhost", "db1", "postgresql://user:pass@localhost/db1"},
		{"postgresql://user@localhost/old?sslmode=disable", "foo", "postgresql://user@localhost/foo?sslmode=disable"},
		{"user=foo dbname=old sslmode=disable", "bar", "user=foo dbname=bar sslmode=disable"},
		{"user=foo", "bar", "user=foo dbname=bar"},
	}

	for _, tt := range tests {
		got := getDSN(tt.dsn, tt.db)
		if got != tt.expect {
			t.Errorf("getDSN(%q, %q)=%q want %q", tt.dsn, tt.db, got, tt.expect)
		}
	}
}

func TestRunSQL(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	receivedDSN := ""
	oldOpen := openDB
	openDB = func(driverName, dsn string) (*sql.DB, error) {
		receivedDSN = dsn
		return db, nil
	}
	defer func() { openDB = oldOpen }()

	mock.ExpectPing()
	mock.ExpectExec("CREATE TABLE test").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO test values\\(1\\)").WillReturnResult(sqlmock.NewResult(1, 1))

	conf := &Config{DSN: "postgresql://localhost", Timeout: 1}
	sqlText := "CREATE TABLE test;\nINSERT INTO test values(1);"

	if err := runSQL(context.Background(), conf, "foo", sqlText); err != nil {
		t.Fatalf("runSQL returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
	expectedDSN := "postgresql://localhost/foo"
	if receivedDSN != expectedDSN {
		t.Errorf("DSN used %q want %q", receivedDSN, expectedDSN)
	}
}
