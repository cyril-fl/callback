// Package liststatuschanges gives the status history of every application, oldest first: the dashboard's funnel and
// response delays come from it.
package liststatuschanges

import (
	"context"

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
	result []domain.StatusChange
}

func Process(ctx context.Context, db bun.IDB) *run {
	return &run{
		ctx: ctx,
		db:  db,
	}
}

func (p *run) Run() {
	p.loadChanges()
}

// Getters & Setters
func (p *run) GetResult() []domain.StatusChange {
	return p.cache.result
}

// Methods

// loadChanges reads them all: a few per application, a few thousand for years of applications.
// shortcut: no filter nor paging, add the applications' filter when the history grows past what a page reads at once.
func (p *run) loadChanges() {
	p.SetErr(p.db.NewSelect().Model(&p.cache.result).OrderExpr("sc.at, sc.id").Scan(p.ctx))
}
