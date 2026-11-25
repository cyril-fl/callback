// Package listtokens lists the API tokens and OAuth connections, newest first: not the sessions.
package listtokens

import (
	"context"

	"github.com/cyril-fl/jat/core/domain"
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
	result []domain.Token
}

func Process(ctx context.Context, db bun.IDB) *run {
	return &run{
		ctx: ctx,
		db:  db,
	}
}

func (p *run) Run() {
	p.loadTokens()
}

// Getters & Setters
func (p *run) GetResult() []domain.Token {
	return p.cache.result
}

// Methods
func (p *run) loadTokens() {
	err := p.db.NewSelect().Model(&p.cache.result).Where("kind != ?", domain.TokenSession).Order("created_at DESC").Scan(p.ctx)
	p.SetErr(err)
}
