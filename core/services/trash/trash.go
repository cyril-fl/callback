// Package trash moves deleted applications aside for a day, so a delete can be undone, then forgets them with their
// history. The reads never see them: they read applications, and the history through its application.
package trash

import (
	"context"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/uptrace/bun"
)

// Keep is how long a deleted application can be restored.
const Keep = 24 * time.Hour

// Throw moves the applications of picked, a SELECT of their ids, to the trash, and returns their ids.
func Throw(ctx context.Context, tx bun.IDB, picked *bun.SelectQuery, now time.Time) ([]string, error) {
	var applications []domain.Application
	if err := tx.NewSelect().Model(&applications).Where("id IN (?)", picked).Scan(ctx); err != nil || len(applications) == 0 {
		return nil, err
	}
	deleted := make([]domain.DeletedApplication, len(applications))
	ids := make([]string, len(applications))
	for i, application := range applications {
		deleted[i] = domain.DeletedApplication{ID: application.ID, Application: application, DeletedAt: now}
		ids[i] = application.ID
	}
	if _, err := tx.NewInsert().Model(&deleted).Exec(ctx); err != nil {
		return nil, err
	}
	_, err := tx.NewDelete().Model((*domain.Application)(nil)).Where("id IN (?)", bun.List(ids)).Exec(ctx)
	return ids, err
}

// Restore moves the applications of ids back from the trash, and returns the ids it found.
func Restore(ctx context.Context, tx bun.IDB, ids []string) ([]string, error) {
	var deleted []domain.DeletedApplication
	if err := tx.NewSelect().Model(&deleted).Where("id IN (?)", bun.List(ids)).Scan(ctx); err != nil || len(deleted) == 0 {
		return nil, err
	}
	applications := make([]domain.Application, len(deleted))
	restored := make([]string, len(deleted))
	for i, d := range deleted {
		applications[i] = d.Application
		restored[i] = d.ID
	}
	if _, err := tx.NewInsert().Model(&applications).Exec(ctx); err != nil {
		return nil, err
	}
	_, err := tx.NewDelete().Model((*domain.DeletedApplication)(nil)).Where("id IN (?)", bun.List(restored)).Exec(ctx)
	return restored, err
}

// Empty forgets for good the applications deleted before before, with their history.
func Empty(ctx context.Context, tx bun.IDB, before time.Time) error {
	expired := tx.NewSelect().Model((*domain.DeletedApplication)(nil)).Column("id").Where("deleted_at < ?", before)
	for _, table := range []string{"status_changes", "follow_ups"} {
		if _, err := tx.NewRaw("DELETE FROM ? WHERE application_id IN (?)", bun.Ident(table), expired).Exec(ctx); err != nil {
			return err
		}
	}
	_, err := tx.NewDelete().Model((*domain.DeletedApplication)(nil)).Where("deleted_at < ?", before).Exec(ctx)
	return err
}
