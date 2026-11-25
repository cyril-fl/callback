// Package purgetokens deletes what expired: sessions, tokens, OAuth connections and codes, login failures.
package purgetokens

import (
	"context"
	"time"

	"github.com/cyril-fl/jat/core/domain"
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
	now := time.Now().UTC()
	p.purge((*domain.Token)(nil), "expires_at IS NOT NULL AND expires_at <= ?", now)
	p.purge((*domain.OAuthCode)(nil), "expires_at <= ?", now)
	p.purge((*domain.LoginFailure)(nil), "expires_at <= ?", now)
}

// Methods
func (p *run) purge(model any, where string, now time.Time) {
	if p.HasErr() {
		return
	}
	_, err := p.db.NewDelete().Model(model).Where(where, now).Exec(p.ctx)
	p.SetErr(err)
}
