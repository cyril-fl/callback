// Package appfilter turns a domain.Filter into the WHERE of a query on applications: the list, the batch update and
// the batch delete pick the same rows for the same filter.
package appfilter

import (
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
			q = q.Where("a.source = ?", filter.Source)
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
			q = q.Where("a.updated_at < ?", filter.UntouchedTime())
		}
		return q
	}
}
