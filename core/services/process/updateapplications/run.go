// Package updateapplications changes applications in one transaction: each op picks applications by ids or filters
// and sets fields. An application the change makes invalid is listed with its error, the others are written.
package updateapplications

import (
	"context"
	"errors"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/appfilter"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

var errDryRun = errors.New("dry run")

// Op picks applications by its ids or filters, and sets fields on each.
type Op struct {
	domain.Filter
	Set domain.Changes `json:"set"`
}

type Input struct {
	Ops []Op `json:"ops"`
	// DryRun writes nothing: the result is what the call would write.
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
	p.ensureOps()
	p.runOps()
	p.makeResult()
}

// Getters & Setters
func (p *run) GetResult() domain.BatchResult {
	return p.cache.result
}

// Methods
func (p *run) ensureOps() {
	switch {
	case len(p.input.Ops) == 0:
		p.SetErr(domain.ErrEmptyBatch)
	case len(p.input.Ops) > domain.MaxBatch:
		p.SetErr(domain.ErrLongBatch)
	}
}

// runOps runs every op in one transaction, rolled back on a dry run.
func (p *run) runOps() {
	if p.HasErr() {
		return
	}
	err := p.db.RunInTx(p.ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		now := time.Now().UTC()
		for i, op := range p.input.Ops {
			if err := p.runOp(ctx, tx, i, op, now); err != nil {
				return err
			}
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
	p.cache.result.Count = len(p.cache.result.IDs)
	p.cache.result.DryRun = p.input.DryRun
	p.cache.result.Summarize(p.input.Summary)
}

// runOp writes one op. An op the input makes impossible fails whole, an application it makes invalid fails alone;
// only an error of the base stops the batch.
func (p *run) runOp(ctx context.Context, tx bun.Tx, index int, op Op, now time.Time) error {
	fail := func(id string, err error) {
		p.cache.result.Failed = append(p.cache.result.Failed, domain.BatchFailure{Op: &index, ID: id, Error: err.Error()})
	}
	if err := check(op); err != nil {
		fail("", err)
		return nil
	}

	var applications []domain.Application
	if err := tx.NewSelect().Model(&applications).ApplyQueryBuilder(appfilter.Where(op.Filter)).Scan(ctx); err != nil {
		return err
	}
	for _, id := range missing(op.IDs, applications) {
		fail(id, domain.NotFound("application not found"))
	}
	for _, application := range applications {
		before := application.Status
		op.Set.Apply(&application)
		application.Normalize()
		if err := application.Validate(); err != nil {
			fail(application.ID, err)
			continue
		}
		application.UpdatedAt = now
		if _, err := tx.NewUpdate().Model(&application).WherePK().ExcludeColumn("id", "created_at").Exec(ctx); err != nil {
			return err
		}
		if application.Status != before {
			change := domain.StatusChange{ApplicationID: application.ID, Status: application.Status, At: now}
			if _, err := tx.NewInsert().Model(&change).Exec(ctx); err != nil {
				return err
			}
		}
		p.cache.result.IDs = append(p.cache.result.IDs, application.ID)
	}
	return nil
}

// Helpers

func check(op Op) error {
	if err := op.Filter.Validate(); err != nil {
		return err
	}
	if op.Filter.Empty() {
		return domain.ErrNoFilter
	}
	if op.Set.Empty() {
		return domain.ErrNoChanges
	}
	return nil
}

// missing are the ids no application has.
func missing(ids []string, applications []domain.Application) []string {
	found := map[string]bool{}
	for _, application := range applications {
		found[application.ID] = true
	}
	var result []string
	for _, id := range ids {
		if !found[id] {
			found[id] = true
			result = append(result, id)
		}
	}
	return result
}
