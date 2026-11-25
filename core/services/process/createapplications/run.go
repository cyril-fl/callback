// Package createapplications creates a batch of applications in one transaction: the valid ones are written, the
// others listed with their error.
package createapplications

import (
	"context"
	"time"

	"github.com/cyril-fl/jat/core/domain"
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
	now := time.Now().UTC()
	for i, item := range p.items {
		application := item.Application()
		application.Normalize()
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
	_, err := p.db.NewInsert().Model(&p.cache.valid).Exec(p.ctx)
	p.SetErr(err)
}
