// Package purgeapplications deletes every application and its status history, to start a new job search: the
// password, the preferences, the sessions and the tokens stay.
package purgeapplications

import (
	"context"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/appfilter"
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
	result int
}

func Process(ctx context.Context, db bun.IDB) *run {
	return &run{
		ctx: ctx,
		db:  db,
	}
}

func (p *run) Run() {
	p.purge()
}

// Getters & Setters

// GetResult is how many applications were deleted.
func (p *run) GetResult() int {
	return p.cache.result
}

// Methods

func (p *run) purge() {
	p.SetErr(p.db.RunInTx(p.ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		for _, model := range []any{(*domain.StatusChange)(nil), (*domain.FollowUp)(nil)} {
			if _, err := tx.NewDelete().Model(model).ApplyQueryBuilder(appfilter.History(ctx)).Exec(ctx); err != nil {
				return err
			}
		}
		user := domain.UserID(ctx)
		if _, err := tx.NewDelete().Model((*domain.DeletedApplication)(nil)).Where("user_id = ?", user).Exec(ctx); err != nil {
			return err
		}
		result, err := tx.NewDelete().Model((*domain.Application)(nil)).Where("user_id = ?", user).Exec(ctx)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		p.cache.result = int(count)
		return err
	}))
}
