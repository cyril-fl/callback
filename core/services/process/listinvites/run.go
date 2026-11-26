// Package listinvites gives the invitations still usable, of every admin: neither used nor expired, newest first.
package listinvites

import (
	"context"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/secret"
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
	result []domain.Invite
}

func Process(ctx context.Context, db bun.IDB) *run {
	return &run{
		ctx: ctx,
		db:  db,
	}
}

func (p *run) Run() {
	p.loadInvites()
}

// Getters & Setters
func (p *run) GetResult() []domain.Invite {
	return p.cache.result
}

// Methods
func (p *run) loadInvites() {
	err := p.db.NewSelect().Model(&p.cache.result).Where("used_at IS NULL AND expires_at > ?", time.Now().UTC()).Order("created_at DESC").Scan(p.ctx)
	for i := range p.cache.result {
		// Without the secret, or another one than it was sealed with, the link is lost: the invitation only revokes.
		p.cache.result[i].Token, _ = secret.Open(p.cache.result[i].Token)
	}
	p.SetErr(err)
}
