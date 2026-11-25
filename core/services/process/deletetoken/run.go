// Package deletetoken revokes an API token or an OAuth connection.
package deletetoken

import (
	"context"

	"github.com/cyril-fl/jat/core/domain"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

var ErrNotFound = domain.NotFound("token not found")

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
	p.deleteToken()
}

// Methods
func (p *run) deleteToken() {
	result, err := p.db.NewDelete().Model((*domain.Token)(nil)).Where("id = ? AND kind != ?", p.id, domain.TokenSession).Exec(p.ctx)
	if err != nil {
		p.SetErr(err)
		return
	}
	if deleted, _ := result.RowsAffected(); deleted == 0 {
		p.SetErr(ErrNotFound)
	}
}
