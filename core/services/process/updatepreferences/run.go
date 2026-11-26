// Package updatepreferences changes the preferences of the account of ctx: those left out keep their value.
package updatepreferences

import (
	"context"
	"encoding/json"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/readpreferences"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

var ErrNoUser = domain.NotFound("no such account")

type run struct {
	plumprimitive.Process
	cache

	ctx     context.Context
	db      bun.IDB
	changes domain.PreferenceChanges
}

type cache struct {
	result domain.Preferences
}

func Process(ctx context.Context, db bun.IDB, changes domain.PreferenceChanges) *run {
	return &run{
		ctx:     ctx,
		db:      db,
		changes: changes,
	}
}

func (p *run) Run() {
	p.loadPreferences()
	p.applyChanges()
	p.savePreferences()
}

// Getters & Setters
func (p *run) GetResult() domain.Preferences {
	return p.cache.result
}

// Methods
func (p *run) loadPreferences() {
	current := readpreferences.Process(p.ctx, p.db)
	current.Run()
	if current.HasErr() {
		p.SetErr(current.GetErr())
		return
	}
	p.cache.result = current.GetResult()
}

func (p *run) applyChanges() {
	if p.HasErr() {
		return
	}
	p.changes.Apply(&p.cache.result)
	p.SetErr(p.cache.result.Validate())
}

func (p *run) savePreferences() {
	if p.HasErr() {
		return
	}
	value, err := json.Marshal(p.cache.result)
	if err != nil {
		p.SetErr(err)
		return
	}
	result, err := p.db.NewUpdate().Model((*domain.User)(nil)).Set("preferences = ?", string(value)).Where("id = ?", domain.UserID(p.ctx)).Exec(p.ctx)
	if err != nil {
		p.SetErr(err)
		return
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		p.SetErr(ErrNoUser)
	}
}
