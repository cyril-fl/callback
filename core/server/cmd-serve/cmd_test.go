package cmdserve

import (
	"context"
	"testing"

	"github.com/cyril-fl/callback/config"
	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/testdb"
	"github.com/spf13/pflag"
)

// The single mode refuses a base of several accounts; the multi mode takes it.
func TestCheckAccounts(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	single, multi := domain.ServerMode{Mode: "single"}, domain.ServerMode{Mode: "multi"}
	testdb.Account(t, db, "cyril")
	if err := checkAccounts(ctx, db, single); err != nil {
		t.Errorf("single, one account: %v", err)
	}
	testdb.Account(t, db, "bob")
	if err := checkAccounts(ctx, db, single); err == nil {
		t.Error("single, two accounts: want an error")
	}
	if err := checkAccounts(ctx, db, multi); err != nil {
		t.Errorf("multi, two accounts: %v", err)
	}
}

// The modes come from the flags, else from the environment, and must be known.
func TestServerMode(t *testing.T) {
	flags := func(args ...string) *pflag.FlagSet {
		set := pflag.NewFlagSet("serve", pflag.ContinueOnError)
		set.String("mode", "single", "")
		set.String("add-user-mode", "invitation", "")
		if err := set.Parse(args); err != nil {
			t.Fatal(err)
		}
		return set
	}
	t.Setenv(config.Env("MODE"), "")
	t.Setenv(config.Env("ADD_USER_MODE"), "")
	if mode, err := serverMode(flags()); err != nil || mode != (domain.ServerMode{Mode: "single", AddUserMode: "invitation"}) {
		t.Errorf("defaults: %+v %v", mode, err)
	}
	t.Setenv(config.Env("MODE"), "multi")
	t.Setenv(config.Env("ADD_USER_MODE"), "public")
	if mode, err := serverMode(flags()); err != nil || mode != (domain.ServerMode{Mode: "multi", AddUserMode: "public"}) {
		t.Errorf("environment: %+v %v", mode, err)
	}
	if mode, err := serverMode(flags("--mode", "single")); err != nil || mode.Mode != "single" {
		t.Errorf("flag over environment: %+v %v", mode, err)
	}
	for _, args := range [][]string{{"--mode", "many"}, {"--add-user-mode", "open"}} {
		if _, err := serverMode(flags(args...)); err == nil {
			t.Errorf("%v: want an error", args)
		}
	}
}
