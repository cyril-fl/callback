// Package readstats counts the applications by status, without reading them.
package readstats

import (
	"context"

	"github.com/cyril-fl/jat/core/domain"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

// Stats are the counts of the applications.
type Stats struct {
	Total int `json:"total"`
	// ByStatus has every status, 0 included.
	ByStatus map[domain.Status]int `json:"by_status"`
	// ResponseRate is the share of applications that got past applied, 0 to 1.
	ResponseRate float64 `json:"response_rate"`
}

type run struct {
	plumprimitive.Process
	cache

	ctx context.Context
	db  bun.IDB
}

type cache struct {
	result Stats
}

func Process(ctx context.Context, db bun.IDB) *run {
	return &run{
		ctx: ctx,
		db:  db,
	}
}

func (p *run) Run() {
	p.countStatuses()
}

// Getters & Setters
func (p *run) GetResult() Stats {
	return p.cache.result
}

// Methods
func (p *run) countStatuses() {
	var rows []struct {
		Status domain.Status
		Count  int
	}
	err := p.db.NewSelect().Model((*domain.Application)(nil)).Column("status").ColumnExpr("count(*) AS count").Group("status").Scan(p.ctx, &rows)
	if err != nil {
		p.SetErr(err)
		return
	}
	stats := Stats{ByStatus: map[domain.Status]int{}}
	for _, status := range domain.Statuses {
		stats.ByStatus[status] = 0
	}
	for _, row := range rows {
		stats.ByStatus[row.Status] = row.Count
		stats.Total += row.Count
	}
	if stats.Total > 0 {
		stats.ResponseRate = float64(stats.Total-stats.ByStatus[domain.StatusApplied]) / float64(stats.Total)
	}
	p.cache.result = stats
}
