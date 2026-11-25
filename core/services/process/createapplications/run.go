// Package createapplications creates a batch of applications in one transaction: the valid ones are written, the
// others listed with their error.
package createapplications

import (
	"context"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/appfilter"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type run struct {
	plumprimitive.Process
	cache

	ctx   context.Context
	db    bun.IDB
	items []domain.NewApplication
}

type cache struct {
	valid  []domain.Application
	result domain.BatchResult
}

func Process(ctx context.Context, db bun.IDB, items []domain.NewApplication) *run {
	return &run{
		ctx:   ctx,
		db:    db,
		items: items,
	}
}

func (p *run) Run() {
	p.ensureItems()
	p.makeApplications()
	p.saveApplications()
}

// Getters & Setters
func (p *run) GetResult() domain.BatchResult {
	return p.cache.result
}

// Methods
func (p *run) ensureItems() {
	switch {
	case len(p.items) == 0:
		p.SetErr(domain.ErrEmptyBatch)
	case len(p.items) > domain.MaxBatch:
		p.SetErr(domain.ErrLongBatch)
	}
}

func (p *run) makeApplications() {
	if p.HasErr() {
		return
	}
	known, err := appfilter.KnownSources(p.ctx, p.db)
	if err != nil {
		p.SetErr(err)
		return
	}
	now := time.Now().UTC()
	for i, item := range p.items {
		application := item.Application()
		application.Normalize()
		application.Source = known.Match(application.Source)
		if err := application.Validate(); err != nil {
			p.cache.result.Failed = append(p.cache.result.Failed, domain.BatchFailure{Index: &i, Error: err.Error()})
			continue
		}
		application.ID = uuid.NewString()
		application.CreatedAt, application.UpdatedAt = now, now
		p.cache.valid = append(p.cache.valid, application)
		p.cache.result.IDs = append(p.cache.result.IDs, application.ID)
	}
	p.cache.result.Count = len(p.cache.valid)
}

func (p *run) saveApplications() {
	if p.HasErr() || len(p.cache.valid) == 0 {
		return
	}
	changes := make([]domain.StatusChange, 0, len(p.cache.valid))
	for _, application := range p.cache.valid {
		changes = append(changes, domain.StatusChange{ApplicationID: application.ID, Status: application.Status, At: application.CreatedAt})
	}
	p.SetErr(p.db.RunInTx(p.ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(&p.cache.valid).Exec(ctx); err != nil {
			return err
		}
		_, err := tx.NewInsert().Model(&changes).Exec(ctx)
		return err
	}))
}
