// Package restoreapplications brings deleted applications back from the trash, with their history: the undo of a
// delete, within a day.
package restoreapplications

import (
	"context"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/trash"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

type Input struct {
	IDs []string `json:"ids"`
}

type run struct {
	plumprimitive.Process
	cache

	ctx   context.Context
	db    bun.IDB
	input Input
}

type cache struct {
	result domain.BatchResult
}

func Process(ctx context.Context, db bun.IDB, input Input) *run {
	return &run{
		ctx:   ctx,
		db:    db,
		input: input,
	}
}

func (p *run) Run() {
	p.ensureIDs()
	p.restore()
	p.makeResult()
}

// Getters & Setters
func (p *run) GetResult() domain.BatchResult {
	return p.cache.result
}

// Methods
func (p *run) ensureIDs() {
	switch {
	case len(p.input.IDs) == 0:
		p.SetErr(domain.ErrEmptyBatch)
	case len(p.input.IDs) > domain.MaxBatch:
		p.SetErr(domain.ErrLongBatch)
	}
}

// restore forgets the expired ones first: an application deleted more than a day ago is gone.
func (p *run) restore() {
	if p.HasErr() {
		return
	}
	p.SetErr(p.db.RunInTx(p.ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := trash.Empty(ctx, tx, time.Now().UTC().Add(-trash.Keep)); err != nil {
			return err
		}
		ids, err := trash.Restore(ctx, tx, p.input.IDs)
		p.cache.result.IDs = ids
		return err
	}))
}

func (p *run) makeResult() {
	if p.HasErr() {
		return
	}
	found := map[string]bool{}
	for _, id := range p.cache.result.IDs {
		found[id] = true
	}
	for _, id := range p.input.IDs {
		if !found[id] {
			found[id] = true
			p.cache.result.Failed = append(p.cache.result.Failed, domain.BatchFailure{ID: id, Error: "application not in the trash"})
		}
	}
	p.cache.result.Count = len(p.cache.result.IDs)
}
