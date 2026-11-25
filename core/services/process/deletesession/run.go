// Package deletesession logs this device out: its session token is deleted.
package deletesession

import (
	"context"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/password"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

type run struct {
	plumprimitive.Process

	ctx   context.Context
	db    bun.IDB
	token string
}

func Process(ctx context.Context, db bun.IDB, token string) *run {
	return &run{
		ctx:   ctx,
		db:    db,
		token: token,
	}
}

func (p *run) Run() {
	p.deleteSession()
}

// Methods
func (p *run) deleteSession() {
	_, err := p.db.NewDelete().Model((*domain.Token)(nil)).Where("hash = ? AND kind = ?", password.TokenHash(p.token), domain.TokenSession).Exec(p.ctx)
	p.SetErr(err)
}
