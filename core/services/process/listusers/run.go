// Package listusers gives every account, for an admin: no password hash, no preferences.
package listusers

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
	result []domain.User
}

func Process(ctx context.Context, db bun.IDB) *run {
	return &run{
		ctx: ctx,
		db:  db,
	}
}

func (p *run) Run() {
	p.loadUsers()
}

// Getters & Setters
func (p *run) GetResult() []domain.User {
	return p.cache.result
}

// Methods
func (p *run) loadUsers() {
	err := p.db.NewSelect().Model(&p.cache.result).Column("id", "name", "admin", "created_at").OrderExpr("lower(name)").Scan(p.ctx)
	p.SetErr(err)
}
