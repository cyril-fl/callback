// Package database opens the base: a SQLite file through the pure Go driver (no cgo, the image is scratch), or a
// Postgres base when the DSN is a postgres:// URL.
package database

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	_ "modernc.org/sqlite"
)

// Open opens the base at dsn: a postgres:// or postgresql:// URL, or else the path of a SQLite file, created when
// missing.
func Open(dsn string) (*bun.DB, error) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return Postgres(dsn)
	}
	return open("file:" + url.PathEscape(dsn) + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_txlock=immediate")
}

// Postgres opens the Postgres base at dsn, with the driver's pool.
func Postgres(dsn string) (*bun.DB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	if err := sqldb.Ping(); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("open the base: %w", err)
	}
	return bun.NewDB(sqldb, pgdialect.New()), nil
}

// Memory opens a SQLite base that lives as long as it stays open: one per test.
func Memory() (*bun.DB, error) {
	return open("file::memory:?_pragma=foreign_keys(1)")
}

// open opens a SQLite base on one connection: SQLite writes one at a time anyway, and the pragmas hold for the
// connection they ran on. WAL lets a backup read while the server writes.
func open(dsn string) (*bun.DB, error) {
	sqldb, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	sqldb.SetMaxOpenConns(1)
	// The memory base is gone with its connection: never close an idle one.
	sqldb.SetConnMaxIdleTime(0)
	sqldb.SetConnMaxLifetime(0)
	if err := sqldb.Ping(); err != nil {
		return nil, fmt.Errorf("open the base: %w", err)
	}
	return bun.NewDB(sqldb, sqlitedialect.New()), nil
}
