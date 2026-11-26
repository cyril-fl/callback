// Package readpreferences gives the preferences of the account of ctx, the defaults where none was saved.
package readpreferences

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/cyril-fl/callback/core/domain"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

type run struct {
	plumprimitive.Process
	cache

	ctx context.Context
	db  bun.IDB
}

type cache struct {
	result domain.Preferences
}

func Process(ctx context.Context, db bun.IDB) *run {
	return &run{
		ctx: ctx,
		db:  db,
	}
}

func (p *run) Run() {
	p.loadPreferences()
}

// Getters & Setters
func (p *run) GetResult() domain.Preferences {
	return p.cache.result
}

// Methods
func (p *run) loadPreferences() {
	p.cache.result = domain.DefaultPreferences
	// The raw JSON, not domain.User: it is read over the defaults.
	var value string
	err := p.db.NewSelect().Model((*domain.User)(nil)).Column("preferences").Where("id = ?", domain.UserID(p.ctx)).Scan(p.ctx, &value)
	if errors.Is(err, sql.ErrNoRows) {
		return
	}
	if err != nil {
		p.SetErr(err)
		return
	}
	// Over the defaults: a preference added since keeps its default.
	p.SetErr(json.Unmarshal([]byte(value), &p.cache.result))
}
