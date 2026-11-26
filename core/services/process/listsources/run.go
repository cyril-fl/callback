// Package listsources gives the sources the applications use, the most used first: the form offers them.
package listsources

import (
	"context"

	"github.com/cyril-fl/callback/core/domain"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

// Source is a source and how many applications have it.
type Source struct {
	Source domain.Source `json:"source" swaggertype:"string"`
	Count  int           `json:"count"`
}

type run struct {
	plumprimitive.Process
	cache

	ctx context.Context
	db  bun.IDB
}

type cache struct {
	result []Source
}

func Process(ctx context.Context, db bun.IDB) *run {
	return &run{
		ctx: ctx,
		db:  db,
	}
}

func (p *run) Run() {
	p.loadSources()
}

// Getters & Setters
func (p *run) GetResult() []Source {
	return p.cache.result
}

// Methods
func (p *run) loadSources() {
	err := p.db.NewSelect().Model((*domain.Application)(nil)).
		Column("source").ColumnExpr("count(*) AS count").Where("user_id = ?", domain.UserID(p.ctx)).
		Group("source").OrderExpr("count DESC, source").
		Scan(p.ctx, &p.cache.result)
	p.SetErr(err)
}
