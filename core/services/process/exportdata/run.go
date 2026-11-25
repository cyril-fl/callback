// Package exportdata reads every application, the trash, the status history and the follow-ups into a Backup.
package exportdata

import (
	"context"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

type run struct {
	plumprimitive.Process
	cache

	ctx context.Context
	db  bun.IDB
}

type cache struct {
	result domain.Backup
}

func Process(ctx context.Context, db bun.IDB) *run {
	return &run{
		ctx: ctx,
		db:  db,
	}
}

func (p *run) Run() {
	p.loadBackup()
}

// Getters & Setters
func (p *run) GetResult() domain.Backup {
	return p.cache.result
}

// Methods

// loadBackup reads in one transaction, so a write in between cannot leave a history without its application.
// shortcut: all in memory, fine for a personal tracker; stream it when a base holds more than a few tens of thousands.
func (p *run) loadBackup() {
	backup := domain.Backup{
		Version:       domain.BackupVersion,
		ExportedAt:    time.Now().UTC(),
		Applications:  []domain.Application{},
		Deleted:       []domain.DeletedApplication{},
		StatusChanges: []domain.StatusChange{},
		FollowUps:     []domain.FollowUp{},
	}
	p.SetErr(p.db.RunInTx(p.ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := tx.NewSelect().Model(&backup.Applications).Order("a.created_at", "a.id").Scan(ctx); err != nil {
			return err
		}
		if err := tx.NewSelect().Model(&backup.Deleted).Order("da.deleted_at", "da.id").Scan(ctx); err != nil {
			return err
		}
		if err := tx.NewSelect().Model(&backup.StatusChanges).Order("sc.id").Scan(ctx); err != nil {
			return err
		}
		return tx.NewSelect().Model(&backup.FollowUps).Order("fu.id").Scan(ctx)
	}))
	p.cache.result = backup
}
