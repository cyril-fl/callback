// Package readstats counts the applications by status and by source, without reading them.
package readstats

import (
	"context"

	"github.com/cyril-fl/callback/core/domain"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

// Stats are the counts of the applications.
type Stats struct {
	Total int `json:"total"`
	// ByStatus has every status, 0 included.
	ByStatus map[domain.Status]int `json:"by_status"`
	// BySource has the sources in use: an agent reuses their names rather than make a near double.
	BySource map[domain.Source]int `json:"by_source"`
	// ResponseRate is the share of applications that got past applied, 0 to 1; left out without applications.
	ResponseRate *float64 `json:"response_rate,omitempty" binding:"optional"`
	// ResponseRateBySource is ResponseRate for each source of BySource.
	ResponseRateBySource map[domain.Source]float64 `json:"response_rate_by_source"`
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
	p.count()
}

// Getters & Setters
func (p *run) GetResult() Stats {
	return p.cache.result
}

// Methods
func (p *run) count() {
	var rows []struct {
		Status domain.Status
		Source domain.Source
		Count  int
	}
	err := p.db.NewSelect().Model((*domain.Application)(nil)).Column("status", "source").ColumnExpr("count(*) AS count").Group("status", "source").Scan(p.ctx, &rows)
	if err != nil {
		p.SetErr(err)
		return
	}
	stats := Stats{ByStatus: map[domain.Status]int{}, BySource: map[domain.Source]int{}, ResponseRateBySource: map[domain.Source]float64{}}
	responded := map[domain.Source]int{}
	for _, status := range domain.Statuses {
		stats.ByStatus[status] = 0
	}
	for _, row := range rows {
		stats.ByStatus[row.Status] += row.Count
		stats.BySource[row.Source] += row.Count
		stats.Total += row.Count
		if row.Status != domain.StatusApplied {
			responded[row.Source] += row.Count
		}
	}
	for source, count := range stats.BySource {
		stats.ResponseRateBySource[source] = float64(responded[source]) / float64(count)
	}
	if stats.Total > 0 {
		rate := float64(stats.Total-stats.ByStatus[domain.StatusApplied]) / float64(stats.Total)
		stats.ResponseRate = &rate
	}
	p.cache.result = stats
}
