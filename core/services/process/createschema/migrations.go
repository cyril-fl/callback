package createschema

import "github.com/uptrace/bun"

// migrations are one-time queries for what createschema cannot do (a renamed or retyped column, data to rewrite),
// run once per base. Append only, never edit nor reorder one that has shipped: its id is how a base knows it ran.
var migrations = []migration{
	// The status history started after the applications: each one applied on its day, then at its current status
	// when it moved on, at its last update. Times are stored as bun writes them, in UTC.
	{"status-changes-backfill", `INSERT INTO status_changes (application_id, status, at)
		SELECT id, 'applied', applied_at || ' 00:00:00+00:00' FROM applications
		UNION ALL
		SELECT id, status, updated_at FROM applications WHERE status <> 'applied'`},
}

type migration struct {
	id    string
	query string
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
			_, err = tx.ExecContext(p.ctx, m.query)
			p.SetErr(err)
		}
	}
}
