// Package appfilter turns a domain.Filter into the WHERE of a query on applications: the list, the batch update and
// the batch delete pick the same rows for the same filter.
package appfilter

import (
	"context"
	"strings"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/uptrace/bun"
)

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// Where adds the filter to q. Validate the filter first.
func Where(filter domain.Filter) func(bun.QueryBuilder) bun.QueryBuilder {
	return func(q bun.QueryBuilder) bun.QueryBuilder {
		if len(filter.IDs) > 0 {
			q = q.Where("a.id IN (?)", bun.List(filter.IDs))
		}
		if len(filter.Status) > 0 {
			q = q.Where("a.status IN (?)", bun.List(filter.Status))
		}
		if filter.Source != "" {
			q = q.Where("a.source = ? COLLATE NOCASE", filter.Source)
		}
		if location := strings.TrimSpace(filter.Location); location != "" {
			q = q.Where("a.location = ? COLLATE NOCASE", location)
		}
		if text := strings.TrimSpace(filter.Q); text != "" {
			like := "%" + likeEscaper.Replace(text) + "%"
			q = q.WhereGroup(" AND ", func(q bun.QueryBuilder) bun.QueryBuilder {
				for _, column := range []string{"company", "location", "street", "postal_code", "country", "website", "source", "source_detail", "contact_name", "contact_email", "comment"} {
					q = q.WhereOr("a.? LIKE ? ESCAPE '\\'", bun.Ident(column), like)
				}
				return q
			})
		}
		if filter.AppliedBefore != "" {
			q = q.Where("a.applied_at < ?", filter.AppliedBefore)
		}
		if filter.AppliedAfter != "" {
			q = q.Where("a.applied_at > ?", filter.AppliedAfter)
		}
		if filter.UntouchedSince != "" {
			since := filter.UntouchedTime()
			q = q.Where("(a.followed_up_at IS NULL OR a.followed_up_at < ?)", since).
				Where("NOT EXISTS (SELECT 1 FROM status_changes WHERE application_id = a.id AND at >= ?)", since)
		}
		return q
	}
}

// KnownSources reads the sources in use, for domain.KnownSources.Match: a write reuses their names whatever the case.
func KnownSources(ctx context.Context, db bun.IDB) (domain.KnownSources, error) {
	var sources []domain.Source
	err := db.NewSelect().Model((*domain.Application)(nil)).Distinct().Column("source").Scan(ctx, &sources)
	return domain.NewKnownSources(sources), err
}
