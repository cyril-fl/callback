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

// Where adds the filter to q, on the applications of the account of ctx only. Validate the filter first.
func Where(ctx context.Context, filter domain.Filter) func(bun.QueryBuilder) bun.QueryBuilder {
	return func(q bun.QueryBuilder) bun.QueryBuilder {
		q = q.Where("a.user_id = ?", domain.UserID(ctx))
		if len(filter.IDs) > 0 {
			q = q.Where("a.id IN (?)", bun.List(filter.IDs))
		}
		if len(filter.Status) > 0 {
			q = q.Where("a.status IN (?)", bun.List(filter.Status))
		}
		if filter.Source != "" {
			q = q.Where("lower(a.source) = lower(?)", filter.Source)
		}
		if location := strings.TrimSpace(filter.Location); location != "" {
			q = q.Where("lower(a.location) = lower(?)", location)
		}
		if text := strings.TrimSpace(filter.Q); text != "" {
			// lower() on both sides: LIKE is case-sensitive on Postgres, ASCII-insensitive on SQLite.
			like := "%" + likeEscaper.Replace(text) + "%"
			q = q.WhereGroup(" AND ", func(q bun.QueryBuilder) bun.QueryBuilder {
				for _, column := range []string{"company", "title", "location", "street", "postal_code", "country", "website", "source", "source_detail", "contact_name", "contact_email", "comment"} {
					q = q.WhereOr("lower(a.?) LIKE lower(?) ESCAPE '\\'", bun.Ident(column), like)
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
			// An interview on that day or after is news too, past or still to come: interview_at is a local
			// "2026-01-31T14:30", which sorts after its day.
			q = q.Where("(a.followed_up_at IS NULL OR a.followed_up_at < ?)", since).
				Where("NOT EXISTS (SELECT 1 FROM status_changes WHERE application_id = a.id AND at >= ?)", since).
				Where("(a.interview_at IS NULL OR a.interview_at < ?)", filter.UntouchedSince)
		}
		return q
	}
}

// History scopes a query on status_changes or follow_ups to the account of ctx, through their application, kept or
// in the trash: they have no user_id of their own.
func History(ctx context.Context) func(bun.QueryBuilder) bun.QueryBuilder {
	return func(q bun.QueryBuilder) bun.QueryBuilder {
		user := domain.UserID(ctx)
		return q.Where("application_id IN (SELECT ka.id FROM applications ka WHERE ka.user_id = ? UNION ALL SELECT kd.id FROM deleted_applications kd WHERE kd.user_id = ?)", user, user)
	}
}

// KnownSources reads the sources the account of ctx uses, for domain.KnownSources.Match: a write reuses their names whatever the case.
func KnownSources(ctx context.Context, db bun.IDB) (domain.KnownSources, error) {
	var sources []domain.Source
	err := db.NewSelect().Model((*domain.Application)(nil)).Distinct().Column("source").Where("user_id = ?", domain.UserID(ctx)).Scan(ctx, &sources)
	return domain.NewKnownSources(sources), err
}
