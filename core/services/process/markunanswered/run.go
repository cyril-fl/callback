// Package markunanswered moves to no_reply the applications waiting on the employer without news for some days: no
// follow-up, status change nor interview since, else since the day they applied, as the untouched_since filter. Serve
// runs it every hour with Preferences.NoReplyDays.
package markunanswered

import (
	"context"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/appfilter"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

type run struct {
	plumprimitive.Process
	cache

	ctx  context.Context
	db   bun.IDB
	days int
	now  time.Time
}

type cache struct {
	result int
}

// Process marks the applications untouched for days before now; days 0 marks none.
func Process(ctx context.Context, db bun.IDB, days int, now time.Time) *run {
	return &run{
		ctx:  ctx,
		db:   db,
		days: days,
		now:  now.UTC(),
	}
}

func (p *run) Run() {
	p.markApplications()
}

// Getters & Setters

// GetResult is how many applications moved to no_reply.
func (p *run) GetResult() int {
	return p.cache.result
}

// Methods

// markApplications writes the new status and its change, as an update would, so undo_status takes it back.
// A preference of 0 days never moves anything.
func (p *run) markApplications() {
	if p.days <= 0 {
		return
	}

	filter := domain.Filter{
		Status:         domain.Waiting,
		UntouchedSince: p.now.AddDate(0, 0, -p.days).Format(domain.DateLayout),
	}
	p.SetErr(p.db.RunInTx(p.ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var applications []domain.Application
		if err := tx.NewSelect().Model(&applications).Column("a.id", "a.status").ApplyQueryBuilder(appfilter.Where(ctx, filter)).Scan(ctx); err != nil || len(applications) == 0 {
			return err
		}
		ids := make([]string, len(applications))
		changes := make([]domain.StatusChange, len(applications))
		for i, application := range applications {
			ids[i] = application.ID
			changes[i] = domain.StatusChange{ApplicationID: application.ID, From: application.Status, Status: domain.StatusNoReply, At: p.now}
		}
		_, err := tx.NewUpdate().Model((*domain.Application)(nil)).
			Set("status = ?", domain.StatusNoReply).Set("updated_at = ?", p.now).
			Where("id IN (?)", bun.List(ids)).Exec(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.NewInsert().Model(&changes).Exec(ctx); err != nil {
			return err
		}
		p.cache.result = len(ids)
		return nil
	}))
}
