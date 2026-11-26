// Package testdb gives each test a base of its own, in memory, with the schema. With CALLBACK_TEST_PG set to a Postgres
// URL, the base is a schema of its own in that Postgres instead, dropped at the end of the test.
package testdb

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/database"
	"github.com/cyril-fl/callback/core/services/process/createschema"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// New is an empty base with the schema, closed at the end of t.
func New(t testing.TB) *bun.DB {
	t.Helper()
	db, err := open(t)
	if err != nil {
		t.Fatal(err)
	}
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

// open opens the base of t, closed (and its Postgres schema dropped) at the end of t.
func open(t testing.TB) (*bun.DB, error) {
	dsn := os.Getenv("CALLBACK_TEST_PG")
	if dsn == "" {
		db, err := database.Memory()
		if err == nil {
			t.Cleanup(func() { db.Close() })
		}
		return db, err
	}
	admin, err := database.Postgres(dsn)
	if err != nil {
		return nil, err
	}
	schema := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		admin.Close()
		return nil, err
	}
	t.Cleanup(func() {
		admin.Exec("DROP SCHEMA " + schema + " CASCADE")
		admin.Close()
	})
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("CALLBACK_TEST_PG: %w", err)
	}
	query := u.Query()
	query.Set("search_path", schema)
	u.RawQuery = query.Encode()
	db, err := database.Postgres(u.String())
	if err == nil {
		t.Cleanup(func() { db.Close() }) // runs before the drop: cleanups are last in, first out
	}
	return db, err
}
