// Package deleteapplications deletes the applications of ids or filters in one transaction: they go to the trash for
// the preferences' trash_days, then are gone with their history.
package deleteapplications

import (
	"context"
	"errors"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/appfilter"
	"github.com/cyril-fl/callback/core/services/trash"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

var errDryRun = errors.New("dry run")

type Input struct {
	domain.Filter
	// DryRun deletes nothing: the result is what the call would delete.
	DryRun bool `json:"dry_run,omitempty" binding:"optional"`
	// Summary leaves the ids out of the result: the counts and failures only.
	Summary bool `json:"summary,omitempty" binding:"optional"`
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
	p.ensureFilter()
	p.deleteApplications()
	p.makeResult()
}

// Getters & Setters
func (p *run) GetResult() domain.BatchResult {
	return p.cache.result
}

// Methods

// ensureFilter refuses an empty filter: a delete of everything is never one call away.
func (p *run) ensureFilter() {
	if err := p.input.Filter.Validate(); err != nil {
		p.SetErr(err)
		return
	}
	if p.input.Filter.Empty() {
		p.SetErr(domain.ErrNoFilter)
	}
}

func (p *run) deleteApplications() {
	if p.HasErr() {
		return
	}
	err := p.db.RunInTx(p.ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// SQLite's DELETE has no alias for the filter's a.: the rows are picked by a SELECT.
		picked := tx.NewSelect().Model((*domain.Application)(nil)).Column("id").ApplyQueryBuilder(appfilter.Where(ctx, p.input.Filter))
		now := time.Now().UTC()
		ids, err := trash.Throw(ctx, tx, picked, now)
		if err != nil {
			return err
		}
		p.cache.result.IDs = ids
		if err := trash.Empty(ctx, tx, now); err != nil {
			return err
		}
		if p.input.DryRun {
			return errDryRun
		}
		return nil
	})
	if !errors.Is(err, errDryRun) {
		p.SetErr(err)
	}
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
			p.cache.result.Failed = append(p.cache.result.Failed, domain.BatchFailure{ID: id, Error: "application not found"})
		}
	}
	p.cache.result.Count = len(p.cache.result.IDs)
	p.cache.result.DryRun = p.input.DryRun
	p.cache.result.Summarize(p.input.Summary)
}
