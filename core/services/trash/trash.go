// Package trash moves deleted applications aside for the preferences' trash_days, so a delete can be undone, then
// forgets them with their history. The reads never see them: they read applications, and the history through its application.
package trash

import (
	"context"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/readpreferences"
	"github.com/uptrace/bun"
)

// Since is the oldest deletion still kept at now: before it, the trash is emptied.
func Since(ctx context.Context, db bun.IDB, now time.Time) (time.Time, error) {
	preferences := readpreferences.Process(ctx, db)
	if preferences.Run(); preferences.HasErr() {
		return time.Time{}, preferences.GetErr()
	}
	return now.AddDate(0, 0, -preferences.GetResult().TrashDays), nil
}

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

// Empty forgets for good the applications deleted before Since, with their history.
func Empty(ctx context.Context, tx bun.IDB, now time.Time) error {
	before, err := Since(ctx, tx, now)
	if err != nil {
		return err
	}
	_, err = Forget(ctx, tx, tx.NewSelect().Model((*domain.DeletedApplication)(nil)).Column("id").Where("deleted_at < ?", before))
	return err
}

// Forget deletes for good the applications of picked, a SELECT of their ids in the trash, with their history, and
// returns their ids.
func Forget(ctx context.Context, tx bun.IDB, picked *bun.SelectQuery) ([]string, error) {
	var ids []string
	if err := tx.NewRaw("DELETE FROM deleted_applications WHERE id IN (?) RETURNING id", picked).Scan(ctx, &ids); err != nil || len(ids) == 0 {
		return nil, err
	}
	for _, table := range []string{"status_changes", "follow_ups"} {
		if _, err := tx.NewRaw("DELETE FROM ? WHERE application_id IN (?)", bun.Ident(table), bun.List(ids)).Exec(ctx); err != nil {
			return nil, err
		}
	}
	return ids, nil
}
