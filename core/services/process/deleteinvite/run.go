// Package deleteinvite revokes an invitation, of any admin.
package deleteinvite

import (
	"context"

	"github.com/cyril-fl/callback/core/domain"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

var ErrNoInvite = domain.NotFound("no such invitation")

type run struct {
	plumprimitive.Process

	ctx context.Context
	db  bun.IDB
	id  string
}

func Process(ctx context.Context, db bun.IDB, id string) *run {
	return &run{
		ctx: ctx,
		db:  db,
		id:  id,
	}
}

func (p *run) Run() {
	p.deleteInvite()
}

// Methods
func (p *run) deleteInvite() {
	result, err := p.db.NewDelete().Model((*domain.Invite)(nil)).Where("id = ?", p.id).Exec(p.ctx)
	if err != nil {
		p.SetErr(err)
		return
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		p.SetErr(ErrNoInvite)
	}
}
