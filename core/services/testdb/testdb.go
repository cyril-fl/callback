// Package testdb gives each test a base of its own, in memory, with the schema.
package testdb

import (
	"context"
	"testing"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/createschema"
	"github.com/cyril-fl/callback/core/services/sqlite"
	"github.com/google/uuid"
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

// Account creates the account name in db and returns a ctx acting for it. No real password: hashing one would only
// slow the tests down; setpassword sets one.
func Account(t testing.TB, db bun.IDB, name string) context.Context {
	t.Helper()
	user := domain.User{ID: uuid.NewString(), Name: name, PasswordHash: "-", CreatedAt: time.Now().UTC()}
	if _, err := db.NewInsert().Model(&user).Exec(context.Background()); err != nil {
		t.Fatalf("test account: %v", err)
	}
	return domain.WithUser(context.Background(), user.ID)
}
