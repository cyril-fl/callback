package cmdadmin_test

import (
	"context"
	"testing"

	"github.com/cyril-fl/callback/config"
	"github.com/cyril-fl/callback/core/domain"
	cmdadmin "github.com/cyril-fl/callback/core/server/cmd-admin"
	"github.com/cyril-fl/callback/core/services/process/createsession"
	"github.com/cyril-fl/callback/core/services/testdb"
)

// The account of an empty base comes from the environment; CALLBACK_USER renames the one account later, never one
// of several.
func TestEnsurePassword(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	names := func() []string {
		var names []string
		if err := db.NewSelect().Model((*domain.User)(nil)).Column("name").Order("name").Scan(ctx, &names); err != nil {
			t.Fatal(err)
		}
		return names
	}

	t.Setenv(config.Env("PASSWORD"), "")
	if err := cmdadmin.EnsurePassword(ctx, db, false); err == nil {
		t.Error("no password: want an error")
	}
	if err := cmdadmin.EnsurePassword(ctx, db, true); err != nil || len(names()) != 0 {
		t.Errorf("multi, no password: %v %v, want the base waiting for its first sign-up", names(), err)
	}
	t.Setenv(config.Env("PASSWORD"), "correct horse")
	t.Setenv(config.Env("USER"), "")
	if err := cmdadmin.EnsurePassword(ctx, db, false); err != nil || len(names()) != 1 || names()[0] != cmdadmin.DefaultUser {
		t.Fatalf("created: %v %v", names(), err)
	}
	if _, err := createsession.CheckPassword(ctx, db, "", "correct horse", false); err != nil {
		t.Errorf("password: %v", err)
	}
	t.Setenv(config.Env("USER"), "cyril")
	if err := cmdadmin.EnsurePassword(ctx, db, false); err != nil || names()[0] != "cyril" {
		t.Errorf("renamed: %v %v", names(), err)
	}
	testdb.Account(t, db, "bob")
	t.Setenv(config.Env("USER"), "zoe")
	if err := cmdadmin.EnsurePassword(ctx, db, false); err != nil || len(names()) != 2 || names()[1] != "cyril" {
		t.Errorf("two accounts: %v %v", names(), err)
	}
}
