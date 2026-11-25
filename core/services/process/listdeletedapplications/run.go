// Package listdeletedapplications lists the trash: the applications deleted within the preferences' trash_days, the
// last deleted first.
package listdeletedapplications

import (
	"context"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/trash"
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
	result []domain.DeletedApplication
}

func Process(ctx context.Context, db bun.IDB) *run {
	return &run{
		ctx: ctx,
		db:  db,
	}
}

func (p *run) Run() {
	p.loadDeleted()
}

// Getters & Setters
func (p *run) GetResult() []domain.DeletedApplication {
	return p.cache.result
}

// Methods

// loadDeleted leaves out the expired ones, which the next delete or restore forgets: a read writes nothing.
// shortcut: no paging, add it if a trash ever holds more than a page reads at once.
func (p *run) loadDeleted() {
	since, err := trash.Since(p.ctx, p.db, time.Now().UTC())
	if err != nil {
		p.SetErr(err)
		return
	}
	p.cache.result = []domain.DeletedApplication{}
	p.SetErr(p.db.NewSelect().Model(&p.cache.result).Where("deleted_at >= ?", since).Order("deleted_at DESC").Scan(p.ctx))
}
