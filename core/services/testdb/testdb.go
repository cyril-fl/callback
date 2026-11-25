// Package testdb gives each test a base of its own, in memory, with the schema.
package testdb

import (
	"context"
	"testing"

	"github.com/cyril-fl/callback/core/services/process/createschema"
	"github.com/cyril-fl/callback/core/services/sqlite"
	"github.com/uptrace/bun"
)

// New is an empty base with the schema, closed at the end of t.
func New(t testing.TB) *bun.DB {
	t.Helper()
	db, err := sqlite.Memory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	schema := createschema.Process(context.Background(), db)
	schema.Run()
	if schema.HasErr() {
		t.Fatalf("create schema: %v", schema.GetErr())
	}
	return db
}
