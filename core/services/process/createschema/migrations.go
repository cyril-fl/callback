package createschema

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
)

// migrations are one-time queries for what createschema cannot do (a renamed or retyped column, data to rewrite),
// run once per base. Append only, never edit nor reorder one that has shipped: its id is how a base knows it ran.
// Those marked sqliteOnly rewrite data of the SQLite era: a Postgres base is always newer, it claims them without
// running them (their SQL is SQLite's: text times, sqlite_master).
var migrations = []migration{
	// The status history started after the applications: each one applied on its day, then at its current status
	// when it moved on, at its last update. Times are stored as bun writes them, in UTC.
	{id: "status-changes-backfill", sqliteOnly: true, query: `INSERT INTO status_changes (application_id, status, at)
		SELECT id, 'applied', applied_at || ' 00:00:00+00:00' FROM applications
		UNION ALL
		SELECT id, status, updated_at FROM applications WHERE status <> 'applied'`},
	// The first change of an application was dated when it was entered: it moves back to the day it applied.
	{id: "status-changes-applied-at", sqliteOnly: true, query: `UPDATE status_changes SET at = (SELECT applied_at || ' 00:00:00+00:00' FROM applications a WHERE a.id = status_changes.application_id)
		WHERE id = (SELECT MIN(id) FROM status_changes s WHERE s.application_id = status_changes.application_id)
		AND at > (SELECT applied_at || ' 00:00:00+00:00' FROM applications a WHERE a.id = status_changes.application_id)`},
	// The one account of the settings becomes a user, admin, who owns every row.
	{id: "users-from-settings", sqliteOnly: true, run: usersFromSettings},
	// Countries were free text: they become codes of domain.Countries, an unknown one goes to the comment.
	{id: "country-codes", run: countryCodes},
	// An application had one interview and one contact: they start the lists, in the trash too.
	{id: "interviews-contacts", run: interviewsContacts},
}

// migration is a query, or run when SQL alone cannot do it.
type migration struct {
	id    string
	query string
	run   func(ctx context.Context, tx bun.Tx) error
	// sqliteOnly is claimed but not run on Postgres.
	sqliteOnly bool
}

// runMigrations runs, in the schema's transaction, the migrations this base has not run yet.
func (p *run) runMigrations(tx bun.Tx) {
	if p.HasErr() {
		return
	}
	_, err := tx.ExecContext(p.ctx, `CREATE TABLE IF NOT EXISTS migrations (id TEXT PRIMARY KEY, applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)`)
	p.SetErr(err)
	for _, m := range migrations {
		if p.HasErr() {
			return
		}
		result, err := tx.ExecContext(p.ctx, `INSERT INTO migrations (id) VALUES (?) ON CONFLICT DO NOTHING`, m.id)
		if err != nil {
			p.SetErr(err)
			return
		}
		if claimed, _ := result.RowsAffected(); claimed == 1 && !(m.sqliteOnly && tx.Dialect().Name() != dialect.SQLite) {
			if m.run != nil {
				p.SetErr(m.run(p.ctx, tx))
				continue
			}
			_, err = tx.ExecContext(p.ctx, m.query)
			p.SetErr(err)
		}
	}
}

// usersFromSettings makes the account of a base older than the users: its password hash and preferences were settings.
// A base without them (a new one) has nothing to move.
func usersFromSettings(ctx context.Context, tx bun.Tx) error {
	var exists bool
	if err := tx.NewRaw(`SELECT count(*) > 0 FROM sqlite_master WHERE type = 'table' AND name = 'settings'`).Scan(ctx, &exists); err != nil || !exists {
		return err
	}
	var hash string
	err := tx.NewRaw(`SELECT value FROM settings WHERE key = 'password_hash'`).Scan(ctx, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	id := uuid.NewString()
	if _, err := tx.ExecContext(ctx, `INSERT INTO users (id, name, password_hash, admin, preferences, created_at)
		VALUES (?, 'admin', ?, 1, COALESCE((SELECT value FROM settings WHERE key = 'preferences'), '{}'), ?)`, id, hash, time.Now().UTC()); err != nil {
		return err
	}
	for _, table := range []string{"applications", "deleted_applications", "tokens", "oauth_codes"} {
		if _, err := tx.ExecContext(ctx, `UPDATE ? SET user_id = ? WHERE user_id = ''`, bun.Ident(table), id); err != nil {
			return err
		}
	}
	return nil
}

// countryCodes stores each country as its code, in the applications, the trash and the default of the preferences.
func countryCodes(ctx context.Context, tx bun.Tx) error {
	var applications []domain.Application
	if err := tx.NewSelect().Model(&applications).Column("id", "country", "comment").Where("country IS NOT NULL").Scan(ctx); err != nil {
		return err
	}
	for _, a := range applications {
		if a.KeepCountry() {
			if _, err := tx.NewUpdate().Model(&a).Column("country", "comment").WherePK().Exec(ctx); err != nil {
				return err
			}
		}
	}
	var deleted []domain.DeletedApplication
	if err := tx.NewSelect().Model(&deleted).Scan(ctx); err != nil {
		return err
	}
	for _, d := range deleted {
		if d.Application.KeepCountry() {
			if _, err := tx.NewUpdate().Model(&d).Column("application").WherePK().Exec(ctx); err != nil {
				return err
			}
		}
	}
	// The stored JSON, not the struct: it holds only what was saved, read over the defaults.
	var users []struct {
		ID          string
		Preferences string
	}
	if err := tx.NewRaw(`SELECT id, preferences FROM users`).Scan(ctx, &users); err != nil {
		return err
	}
	for _, u := range users {
		var preferences map[string]any
		if err := json.Unmarshal([]byte(u.Preferences), &preferences); err != nil {
			return err
		}
		country, _ := preferences["default_country"].(string)
		code := domain.CountryCode(country)
		if !domain.IsCountry(code) {
			code = "" // an unknown default is only a preference: it is cleared
		}
		if code == country {
			continue
		}
		preferences["default_country"] = code
		stored, err := json.Marshal(preferences)
		if err != nil {
			return err
		}
		if _, err := tx.NewUpdate().Table("users").Set("preferences = ?", string(stored)).Where("id = ?", u.ID).Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

// interviewsContacts makes the lists of the applications, and of the trash, from interview_at and the contact.
func interviewsContacts(ctx context.Context, tx bun.Tx) error {
	var applications []domain.Application
	if err := tx.NewSelect().Model(&applications).Column("id", "interview_at", "contact_name", "contact_email").Scan(ctx); err != nil {
		return err
	}
	var deleted []domain.DeletedApplication
	if err := tx.NewSelect().Model(&deleted).Scan(ctx); err != nil {
		return err
	}
	for _, d := range deleted {
		d.Application.ID = d.ID
		applications = append(applications, d.Application)
	}
	var interviews []domain.Interview
	var contacts []domain.Contact
	for _, a := range applications {
		if a.InterviewAt != "" {
			interviews = append(interviews, domain.Interview{ApplicationID: a.ID, At: a.InterviewAt, Kind: domain.InterviewOther})
		}
		if a.ContactName != "" || a.ContactEmail != "" {
			contacts = append(contacts, domain.Contact{ApplicationID: a.ID, Name: a.ContactName, Email: a.ContactEmail})
		}
	}
	if len(interviews) > 0 {
		if _, err := tx.NewInsert().Model(&interviews).Exec(ctx); err != nil {
			return err
		}
	}
	if len(contacts) > 0 {
		_, err := tx.NewInsert().Model(&contacts).Exec(ctx)
		return err
	}
	return nil
}
