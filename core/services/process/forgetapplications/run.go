// Package forgetapplications deletes applications from the trash for good, with their history: some by ids, or all.
package forgetapplications

import (
	"context"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/trash"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

var ErrNoIDs = domain.Invalid("give ids, or all to empty the trash")

type Input struct {
	IDs []string `json:"ids,omitempty" binding:"optional"`
	// All empties the trash; without it, ids are required.
	All bool `json:"all,omitempty" binding:"optional"`
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
	p.ensureInput()
	p.forget()
	p.makeResult()
}

// Getters & Setters
func (p *run) GetResult() domain.BatchResult {
	return p.cache.result
}

// Methods

// ensureInput refuses a call without ids nor all: emptying the trash is never a body left out.
func (p *run) ensureInput() {
	switch {
	case p.input.All == (len(p.input.IDs) > 0):
		p.SetErr(ErrNoIDs)
	case len(p.input.IDs) > domain.MaxBatch:
		p.SetErr(domain.ErrLongBatch)
	}
}

func (p *run) forget() {
	if p.HasErr() {
		return
	}
	p.SetErr(p.db.RunInTx(p.ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// trash.Forget keeps to the account.
		picked := tx.NewSelect().Model((*domain.DeletedApplication)(nil)).Column("id")
		if !p.input.All {
			picked = picked.Where("id IN (?)", bun.List(p.input.IDs))
		}
		ids, err := trash.Forget(ctx, tx, picked)
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
