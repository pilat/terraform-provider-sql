package sql

import (
	"context"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"

	sqlDB "database/sql"

	_ "github.com/lib/pq"
)

var openDB = sqlDB.Open

func getDSN(dsn, dbName string) string {
	if strings.HasPrefix(dsn, "postgresql://") {
		parsedURL, err := url.Parse(dsn)
		if err != nil {
			log.Fatal(err)
		}

		parsedURL.Path = dbName

		return parsedURL.String()
	}

	found := false
	parts := strings.Split(dsn, " ")
	for i, part := range parts {
		if strings.HasPrefix(part, "dbname=") {
			found = true
			parts[i] = "dbname=" + dbName
			break
		}
	}

	if !found {
		parts = append(parts, "dbname="+dbName)
	}

	return strings.Join(parts, " ")
}

func runSQL(ctx context.Context, conf *Config, dbName, sql string) error {
	db, err := openDB("postgres", getDSN(conf.DSN, dbName))
	if err != nil {
		return err
	}
	defer db.Close()

	db.SetConnMaxLifetime(time.Duration(conf.Timeout) * time.Second)

	if err = db.PingContext(ctx); err != nil {
		return err
	}

	r := regexp.MustCompile(`;(?s)(\r?\n|$)`)
	for _, command := range r.Split(sql, -1) {
		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}

		if _, err = db.ExecContext(ctx, command); err != nil {
			return err
		}
	}

	return nil
}
