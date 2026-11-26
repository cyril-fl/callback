package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cyril-fl/callback/cli/services/apiclient"
	"github.com/cyril-fl/callback/cli/services/apiclient/apitest"
	"github.com/cyril-fl/callback/config"
	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/testserver"
)

// run runs callback with args and returns its stdout.
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	return apitest.Stdout(t, func() error {
		root := newRoot()
		root.SetArgs(args)
		return root.Execute()
	})
}

func must(t *testing.T, args ...string) string {
	t.Helper()
	out, err := run(t, args...)
	if err != nil {
		t.Fatalf("callback %s: %v", strings.Join(args, " "), err)
	}
	return out
}

// TestApplications walks the commands on the applications against the real API.
func TestApplications(t *testing.T) {
	apitest.Serve(t)

	var added apiclient.DomainBatchResult
	json.Unmarshal([]byte(must(t, "add", "--company", "Acme", "--location", "Paris", "--applied-at", "2026-01-02", "--json")), &added)
	file := filepath.Join(t.TempDir(), "apps.json")
	os.WriteFile(file, []byte(`[{"company":"Globex","location":"Lyon","applied_at":"2026-01-03"},{"company":"","location":"x"}]`), 0o600)
	out, err := run(t, "add", "--file", file)
	if err == nil || !strings.HasPrefix(out, "1 added\n") {
		t.Errorf("add --file with a failure = %q, %v; want 1 added and an error", out, err)
	}
	if added.Ids == nil || len(*added.Ids) != 1 {
		t.Fatalf("add = %+v", added)
	}
	acme := (*added.Ids)[0]

	if out := must(t, "ls", "--status", "applied"); !strings.Contains(out, "ID") || !strings.Contains(out, "Acme") || !strings.Contains(out, "Globex") {
		t.Errorf("ls = %q", out)
	}
	if out := must(t, "ls", "-q", "acme", "--count"); out != "1\n" {
		t.Errorf("ls --count = %q", out)
	}
	if out := must(t, "edit", "--where-status", "applied", "-q", "lyon", "--status", "interview", "--dry-run"); !strings.HasPrefix(out, "1 would be updated") {
		t.Errorf("edit --dry-run = %q", out)
	}
	must(t, "edit", acme, "--status", "rejected", "--rating", "2")
	if out := must(t, "show", acme); !strings.Contains(out, "status: rejected") || !strings.Contains(out, "rating: 2") {
		t.Errorf("show = %q", out)
	}
	var stats apiclient.ReadstatsStats
	json.Unmarshal([]byte(must(t, "stats", "--json")), &stats)
	if stats.Total != 2 || stats.ByStatus["rejected"] != 1 {
		t.Errorf("stats = %+v", stats)
	}
	if out := must(t, "rm", "--status", "rejected"); !strings.HasPrefix(out, "1 deleted") {
		t.Errorf("rm = %q", out)
	}
	if _, err := run(t, "show", acme); err == nil {
		t.Error("show of a deleted application: no error")
	}
}

// TestCustomSourceAndAddress adds an application with a source of the user's own and a full address, and reads them back.
func TestCustomSourceAndAddress(t *testing.T) {
	apitest.Serve(t)
	must(t, "add", "--company", "Adecco", "--location", "Lyon", "--street", "1 rue X", "--postal-code", "69001", "--country", "France",
		"--source", "Agence intérim Adecco", "--source-detail", "Marie")
	var rows []map[string]any
	if err := json.Unmarshal([]byte(must(t, "ls", "--fields", "source,country", "--json", "--source", "Agence intérim Adecco")), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["source"] != "Agence intérim Adecco" || rows[0]["country"] != "FR" {
		t.Errorf("ls --fields source,country = %v", rows)
	}
}

// TestAccount logs in, makes a token, and logs out.
func TestAccount(t *testing.T) {
	server := apitest.Serve(t)
	t.Setenv(config.Env("URL"), "")
	t.Setenv(config.Env("SESSION"), "")
	t.Setenv(config.Env("PASSWORD"), testserver.Password)
	must(t, "login", server.URL)
	file, _ := apiclient.SessionFile()
	if info, err := os.Stat(file); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("session file: %v, %v", info, err)
	}
	if out := must(t, "whoami"); !strings.HasPrefix(out, "session on "+server.URL) {
		t.Errorf("whoami = %q", out)
	}
	token := strings.TrimSpace(must(t, "token", "add", "reader", "--read-only"))
	if !strings.HasPrefix(token, "callback_") {
		t.Errorf("token add = %q", token)
	}
	if out := must(t, "token", "ls"); !strings.Contains(out, "reader") {
		t.Errorf("token ls = %q", out)
	}
	must(t, "logout")
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("session file after logout: %v", err)
	}
	if _, err := run(t, "whoami"); err != apiclient.ErrNoSession {
		t.Errorf("whoami after logout: %v", err)
	}
}

// TestLoginMulti logs in to a server of several accounts: the name from --user, refused without it off a terminal.
func TestLoginMulti(t *testing.T) {
	server := testserver.ServeMode(t, domain.ServerMode{Mode: domain.Modes[1], AddUserMode: domain.AddUserModes[0]})
	t.Setenv("HOME", t.TempDir())
	t.Setenv(config.Env("URL"), "")
	t.Setenv(config.Env("SESSION"), "")
	t.Setenv(config.Env("PASSWORD"), testserver.Password)
	stdin, _ := os.Open(os.DevNull)
	defer stdin.Close()
	before := os.Stdin
	os.Stdin = stdin
	defer func() { os.Stdin = before }()

	if _, err := run(t, "login", server.URL); err == nil || !strings.Contains(err.Error(), "--user") {
		t.Errorf("login without --user off a terminal: %v", err)
	}
	if _, err := run(t, "login", server.URL, "--user", "nobody"); err == nil {
		t.Error("login with a wrong user: no error")
	}
	must(t, "login", server.URL, "--user", "admin")
	if out := must(t, "whoami"); !strings.HasPrefix(out, "session of admin (admin) on "+server.URL) {
		t.Errorf("whoami = %q", out)
	}
	var me apiclient.LoginMeResult
	json.Unmarshal([]byte(must(t, "whoami", "--json")), &me)
	if me.User != "admin" || !me.Admin || me.Mode != apiclient.LoginMeResultModeMulti {
		t.Errorf("whoami --json = %+v", me)
	}
}
