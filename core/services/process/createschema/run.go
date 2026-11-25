// Package createschema creates what the base lacks, at every start: tables, indexes, the columns a model gained, then
// the one-time migrations. There is no migration tool.
package createschema

import (
	"context"

	"github.com/cyril-fl/callback/core/domain"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

type run struct {
	plumprimitive.Process

	ctx context.Context
	db  bun.IDB
}

func Process(ctx context.Context, db bun.IDB) *run {
	return &run{
		ctx: ctx,
		db:  db,
	}
}

func (p *run) Run() {
	p.SetErr(p.db.RunInTx(p.ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		p.createTables(tx)
		p.createIndexes(tx)
		p.runMigrations(tx)
		return p.GetErr()
	}))
}

// Methods
func (p *run) createTables(tx bun.Tx) {
	for _, model := range models {
		if p.HasErr() {
			return
		}
		_, err := tx.NewCreateTable().Model(model).IfNotExists().Exec(p.ctx)
		p.SetErr(err)
		p.addColumns(tx, model)
	}
}

// addColumns gives an existing table the columns its model gained since: CREATE TABLE IF NOT EXISTS skips it whole.
// A new NOT NULL column needs a default, SQLite refuses it otherwise.
func (p *run) addColumns(tx bun.Tx, model any) {
	if p.HasErr() {
		return
	}
	table := tx.NewCreateTable().Model(model).GetModel().(bun.TableModel).Table()
	var existing []struct {
		Name string `bun:"name"`
	}
	if err := tx.NewRaw("SELECT name FROM pragma_table_info(?)", table.Name).Scan(p.ctx, &existing); err != nil {
		p.SetErr(err)
		return
	}
	have := map[string]bool{}
	for _, column := range existing {
		have[column.Name] = true
	}
	for _, field := range table.Fields {
		if have[field.Name] || p.HasErr() {
			continue
		}
		column := "? " + field.CreateTableSQLType
		if field.NotNull {
			column += " NOT NULL"
		}
		if field.SQLDefault != "" {
			column += " DEFAULT " + field.SQLDefault
		}
		_, err := tx.NewAddColumn().Model(model).ColumnExpr(column, bun.Ident(field.Name)).Exec(p.ctx)
		p.SetErr(err)
	}
}

func (p *run) createIndexes(tx bun.Tx) {
	for _, index := range indexes {
		if p.HasErr() {
			return
		}
		_, err := tx.ExecContext(p.ctx, index)
		p.SetErr(err)
	}
}

// Helpers

var models = []any{
	(*domain.Application)(nil),
	(*domain.DeletedApplication)(nil),
	(*domain.StatusChange)(nil),
	(*domain.FollowUp)(nil),
	(*domain.Setting)(nil),
	(*domain.Token)(nil),
	(*domain.OAuthCode)(nil),
	(*domain.LoginFailure)(nil),
}

var indexes = []string{
	`CREATE INDEX IF NOT EXISTS idx_applications_status ON applications (status)`,
	`CREATE INDEX IF NOT EXISTS idx_applications_applied_at ON applications (applied_at)`,
	`CREATE INDEX IF NOT EXISTS idx_status_changes_application ON status_changes (application_id, at)`,
	`CREATE INDEX IF NOT EXISTS idx_follow_ups_application ON follow_ups (application_id, at)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS ux_tokens_refresh_hash ON tokens (refresh_hash) WHERE refresh_hash IS NOT NULL`,
}
