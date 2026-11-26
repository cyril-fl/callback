// Package liststatuschanges gives the status history of the applications, oldest first: the dashboard's funnel and
// response delays come from it, an agent reads it for some applications or since a day.
package liststatuschanges

import (
	"context"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

var ErrInvalidSince = domain.Invalid("since is a day like 2026-01-31")

// Input narrows the history: the changes of these applications, reached on that day or after. Empty, all of it.
type Input struct {
	IDs   []string
	Since string
}

type run struct {
	plumprimitive.Process
	cache

	ctx   context.Context
	db    bun.IDB
	input Input
}

type cache struct {
	result []domain.StatusChange
}

func Process(ctx context.Context, db bun.IDB, input Input) *run {
	return &run{
		ctx:   ctx,
		db:    db,
		input: input,
	}
}

func (p *run) Run() {
	p.checkSince()
	p.loadChanges()
}

// Getters & Setters
func (p *run) GetResult() []domain.StatusChange {
	return p.cache.result
}

// Methods

func (p *run) checkSince() {
	if p.input.Since != "" && !domain.IsDate(p.input.Since) {
		p.SetErr(ErrInvalidSince)
	}
}

// loadChanges reads the changes asked: a few per application, a few thousand for years of applications.
// shortcut: no paging, add it when the history grows past what a page reads at once.
func (p *run) loadChanges() {
	if p.HasErr() {
		return
	}

	q := p.db.NewSelect().Model(&p.cache.result).ColumnExpr("sc.*, a.company").
		Join("JOIN applications AS a ON a.id = sc.application_id").Where("a.user_id = ?", domain.UserID(p.ctx)).OrderExpr("sc.at, sc.id")
	if len(p.input.IDs) > 0 {
		q = q.Where("sc.application_id IN (?)", bun.In(p.input.IDs))
	}
	if p.input.Since != "" {
		since, _ := time.Parse(domain.DateLayout, p.input.Since)
		q = q.Where("sc.at >= ?", since)
	}
	p.SetErr(q.Scan(p.ctx))
}
