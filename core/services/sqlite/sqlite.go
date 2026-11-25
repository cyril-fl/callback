// Package sqlite opens the base: one SQLite file, through the pure Go driver (no cgo, the image is scratch).
package sqlite

import (
	"database/sql"
	"fmt"
	"net/url"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"
)

// Open opens the base at path, created when missing. One connection: SQLite writes one at a time anyway, and the
// pragmas hold for the connection they ran on. WAL lets a backup read while the server writes.
func Open(path string) (*bun.DB, error) {
	dsn := "file:" + url.PathEscape(path) + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_txlock=immediate"
	return open(dsn)
}

// Memory opens a base that lives as long as it stays open: one per test.
func Memory() (*bun.DB, error) {
	return open("file::memory:?_pragma=foreign_keys(1)")
}

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
