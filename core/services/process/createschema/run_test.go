package createschema_test

import (
	"context"
	"testing"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/createschema"
	"github.com/cyril-fl/callback/core/services/process/readpreferences"
	"github.com/cyril-fl/callback/core/services/testdb"
)

// The schema runs at every start: a second run on a base that has it changes nothing and fails on nothing.
func TestRunTwice(t *testing.T) {
	db := testdb.New(t)
	again := createschema.Process(context.Background(), db)
	again.Run()
	if again.HasErr() {
		t.Fatalf("second run: %v", again.GetErr())
	}
}

// A column a model gained is added to the table of an older base.
func TestAddColumn(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `ALTER TABLE applications DROP COLUMN comment`); err != nil {
		t.Fatal(err)
	}
	again := createschema.Process(ctx, db)
	again.Run()
	if again.HasErr() {
		t.Fatalf("run: %v", again.GetErr())
	}
	var count int
	if err := db.NewRaw(`SELECT count(*) FROM pragma_table_info('applications') WHERE name = 'comment'`).Scan(ctx, &count); err != nil || count != 1 {
		t.Fatalf("comment column: %d, %v", count, err)
	}
}

// The applications of a base older than the history get one: applied on their day, and their status at their last
// update when they moved on.
func TestBackfillStatusChanges(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	updated := time.Date(2026, 2, 3, 14, 30, 0, 0, time.UTC)
	applications := []domain.Application{
		{ID: "a", Company: "Acme", Location: "Paris", Source: domain.SourceSpontaneous, Status: domain.StatusApplied, AppliedAt: "2026-01-10", CreatedAt: updated, UpdatedAt: updated},
		{ID: "b", Company: "Globex", Location: "Lyon", Source: domain.SourceSpontaneous, Status: domain.StatusInterview, AppliedAt: "2026-01-20", CreatedAt: updated, UpdatedAt: updated},
	}
	if _, err := db.NewInsert().Model(&applications).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM migrations WHERE id = 'status-changes-backfill'`); err != nil {
		t.Fatal(err)
	}
	again := createschema.Process(ctx, db)
	again.Run()
	if again.HasErr() {
		t.Fatal(again.GetErr())
	}
	var changes []domain.StatusChange
	if err := db.NewSelect().Model(&changes).OrderExpr("sc.application_id, sc.at").Scan(ctx); err != nil {
		t.Fatal(err)
	}
	want := []domain.StatusChange{
		{ApplicationID: "a", Status: domain.StatusApplied, At: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)},
		{ApplicationID: "b", Status: domain.StatusApplied, At: time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)},
		{ApplicationID: "b", Status: domain.StatusInterview, At: updated},
	}
	if len(changes) != len(want) {
		t.Fatalf("changes = %+v", changes)
	}
	for i, change := range changes {
		if change.ApplicationID != want[i].ApplicationID || change.Status != want[i].Status || !change.At.Equal(want[i].At) {
			t.Errorf("change %d = %+v, want %+v", i, change, want[i])
		}
	}
}

func TestFirstChangeOnAppliedAt(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	entered := time.Date(2026, 2, 3, 14, 30, 0, 0, time.UTC)
	if _, err := db.NewInsert().Model(&domain.Application{ID: "a", Company: "Acme", Location: "Paris", Source: domain.SourceSpontaneous, Status: domain.StatusResponded, AppliedAt: "2026-01-05", CreatedAt: entered, UpdatedAt: entered}).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	// Entered after it applied, its first change dated then: moved back to the day it applied, the later ones kept.
	changes := []domain.StatusChange{{ApplicationID: "a", Status: domain.StatusApplied, At: entered}, {ApplicationID: "a", From: domain.StatusApplied, Status: domain.StatusResponded, At: entered.Add(time.Hour)}}
	if _, err := db.NewInsert().Model(&changes).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM migrations WHERE id = 'status-changes-applied-at'`); err != nil {
		t.Fatal(err)
	}
	again := createschema.Process(ctx, db)
	again.Run()
	if again.HasErr() {
		t.Fatal(again.GetErr())
	}
	changes = nil
	if err := db.NewSelect().Model(&changes).Order("sc.id").Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 || !changes[0].At.Equal(time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)) || !changes[1].At.Equal(entered.Add(time.Hour)) {
		t.Errorf("changes = %+v", changes)
	}
}

// A base of the single account (its password and preferences in settings, rows without user_id) ends with one admin
// account that owns every row and keeps the preferences.
func TestUsersFromSettings(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	for _, query := range []string{
		`DROP INDEX idx_applications_user`,
		`ALTER TABLE applications DROP COLUMN user_id`,
		`ALTER TABLE tokens DROP COLUMN user_id`,
		`CREATE TABLE settings (key VARCHAR NOT NULL PRIMARY KEY, value VARCHAR NOT NULL)`,
		`INSERT INTO settings VALUES ('password_hash', 'the-hash'), ('preferences', '{"plan":"pro","trash_days":12}')`,
		`INSERT INTO applications (id, company, location, website, source, source_detail, status, rating, comment, applied_at, created_at, updated_at) VALUES ('a', 'Acme', 'Paris', '', 'spontaneous', '', 'applied', 0, '', '2026-01-10', '2026-01-10 00:00:00+00:00', '2026-01-10 00:00:00+00:00')`,
		`INSERT INTO tokens (id, kind, name, hash, read_only, created_at) VALUES ('t', 'session', '', 'h', 0, '2026-01-10 00:00:00+00:00')`,
		`DELETE FROM migrations WHERE id = 'users-from-settings'`,
	} {
		if _, err := db.ExecContext(ctx, query); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}
	again := createschema.Process(ctx, db)
	again.Run()
	if again.HasErr() {
		t.Fatal(again.GetErr())
	}
	var users []domain.User
	if err := db.NewSelect().Model(&users).Scan(ctx); err != nil || len(users) != 1 {
		t.Fatalf("users = %+v, %v", users, err)
	}
	user := users[0]
	if user.Name != "admin" || !user.Admin || user.PasswordHash != "the-hash" {
		t.Errorf("user = %+v", user)
	}
	var orphans int
	if err := db.NewRaw(`SELECT (SELECT count(*) FROM applications WHERE user_id <> ?) + (SELECT count(*) FROM tokens WHERE user_id <> ?)`, user.ID, user.ID).Scan(ctx, &orphans); err != nil || orphans != 0 {
		t.Errorf("rows of no one: %d, %v", orphans, err)
	}
	preferences := readpreferences.Process(domain.WithUser(ctx, user.ID), db)
	preferences.Run()
	if got := preferences.GetResult(); preferences.HasErr() || got.Plan != domain.PlanPro || got.TrashDays != 12 || got.FollowUpDays != domain.DefaultPreferences.FollowUpDays {
		t.Errorf("preferences = %+v, %v", got, preferences.GetErr())
	}
}
