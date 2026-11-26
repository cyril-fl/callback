// Package deleteuser deletes an account for good with everything it holds: its applications, its trash, their
// history and follow-ups, its tokens, OAuth codes and the invitations it made. An admin cannot delete themself.
package deleteuser

import (
	"context"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/appfilter"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

var (
	ErrSelf   = domain.Invalid("an admin cannot delete their own account")
	ErrNoUser = domain.NotFound("no such account")
)

type run struct {
	plumprimitive.Process

	ctx    context.Context
	db     bun.IDB
	userID string
}

// Process deletes the account userID, for the admin of ctx.
func Process(ctx context.Context, db bun.IDB, userID string) *run {
	return &run{
		ctx:    ctx,
		db:     db,
		userID: userID,
	}
}

func (p *run) Run() {
	p.ensureOther()
	p.deleteUser()
}

// Methods
func (p *run) ensureOther() {
	if p.userID == domain.UserID(p.ctx) {
		p.SetErr(ErrSelf)
	}
}

func (p *run) deleteUser() {
	if p.HasErr() {
		return
	}
	// The history goes through the applications of the account deleted, not of the admin.
	history := appfilter.History(domain.WithUser(p.ctx, p.userID))
	p.SetErr(p.db.RunInTx(p.ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		for _, model := range []any{(*domain.StatusChange)(nil), (*domain.FollowUp)(nil), (*domain.Interview)(nil), (*domain.Contact)(nil)} {
			if _, err := tx.NewDelete().Model(model).ApplyQueryBuilder(history).Exec(ctx); err != nil {
				return err
			}
		}
		for _, model := range []any{(*domain.DeletedApplication)(nil), (*domain.Application)(nil), (*domain.Token)(nil), (*domain.OAuthCode)(nil), (*domain.Invite)(nil)} {
			if _, err := tx.NewDelete().Model(model).Where("user_id = ?", p.userID).Exec(ctx); err != nil {
				return err
			}
		}
		result, err := tx.NewDelete().Model((*domain.User)(nil)).Where("id = ?", p.userID).Exec(ctx)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows == 0 {
			return ErrNoUser
		}
		return nil
	}))
}
