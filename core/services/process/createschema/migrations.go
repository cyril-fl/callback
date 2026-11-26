package createschema

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// migrations are one-time queries for what createschema cannot do (a renamed or retyped column, data to rewrite),
// run once per base. Append only, never edit nor reorder one that has shipped: its id is how a base knows it ran.
var migrations = []migration{
	// The status history started after the applications: each one applied on its day, then at its current status
	// when it moved on, at its last update. Times are stored as bun writes them, in UTC.
	{id: "status-changes-backfill", query: `INSERT INTO status_changes (application_id, status, at)
		SELECT id, 'applied', applied_at || ' 00:00:00+00:00' FROM applications
		UNION ALL
		SELECT id, status, updated_at FROM applications WHERE status <> 'applied'`},
	// The first change of an application was dated when it was entered: it moves back to the day it applied.
	{id: "status-changes-applied-at", query: `UPDATE status_changes SET at = (SELECT applied_at || ' 00:00:00+00:00' FROM applications a WHERE a.id = status_changes.application_id)
		WHERE id = (SELECT MIN(id) FROM status_changes s WHERE s.application_id = status_changes.application_id)
		AND at > (SELECT applied_at || ' 00:00:00+00:00' FROM applications a WHERE a.id = status_changes.application_id)`},
	// The one account of the settings becomes a user, admin, who owns every row.
	{id: "users-from-settings", run: usersFromSettings},
}

// migration is a query, or run when SQL alone cannot do it.
type migration struct {
	id    string
	query string
	run   func(ctx context.Context, tx bun.Tx) error
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
		if claimed, _ := result.RowsAffected(); claimed == 1 {
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
