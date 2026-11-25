// Package importdata restores a Backup in one transaction. It only adds: an application whose id the base already
// has (live or in the trash) is left as it is, with its history, so importing twice or into a used base is safe.
package importdata

import (
	"context"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/appfilter"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

// Result counts what was added and lists what was refused: Failed holds the applications that are invalid (Index
// is the place in the file's applications), Skipped is how many the base already had.
type Result struct {
	Added   int                   `json:"added"`
	Deleted int                   `json:"deleted"`
	Skipped int                   `json:"skipped"`
	Failed  []domain.BatchFailure `json:"failed,omitempty" binding:"optional"`
}

type run struct {
	plumprimitive.Process
	cache

	ctx    context.Context
	db     bun.IDB
	backup domain.Backup
}

type cache struct {
	result Result
}

func Process(ctx context.Context, db bun.IDB, backup domain.Backup) *run {
	return &run{
		ctx:    ctx,
		db:     db,
		backup: backup,
	}
}

func (p *run) Run() {
	p.ensureBackup()
	p.restore()
}

// Getters & Setters
func (p *run) GetResult() Result {
	return p.cache.result
}

// Methods
func (p *run) ensureBackup() {
	if p.backup.Version != domain.BackupVersion {
		p.SetErr(domain.ErrInvalidBackup)
	}
}

func (p *run) restore() {
	if p.HasErr() {
		return
	}
	p.SetErr(p.db.RunInTx(p.ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		taken, err := takenIDs(ctx, tx)
		if err != nil {
			return err
		}
		known, err := appfilter.KnownSources(ctx, tx)
		if err != nil {
			return err
		}
		added := map[string]bool{}
		var applications []domain.Application
		for i, application := range p.backup.Applications {
			id := application.ID
			application.Normalize()
			application.Source = known.Match(application.Source)
			switch err := application.Validate(); {
			case id == "":
				p.fail(i, domain.Invalid("id is required"))
			case err != nil:
				p.fail(i, err)
			case taken[id] || added[id]:
				p.cache.result.Skipped++
			default:
				added[id] = true
				applications = append(applications, application)
			}
		}
		var deleted []domain.DeletedApplication
		for _, item := range p.backup.Deleted {
			if item.ID == "" || taken[item.ID] || added[item.ID] {
				p.cache.result.Skipped++
				continue
			}
			added[item.ID] = true
			deleted = append(deleted, item)
		}
		if err := insert(ctx, tx, applications); err != nil {
			return err
		}
		if err := insert(ctx, tx, deleted); err != nil {
			return err
		}
		p.cache.result.Added, p.cache.result.Deleted = len(applications), len(deleted)

		// The history of an application added only: the ones the base kept have theirs.
		var changes []domain.StatusChange
		for _, change := range p.backup.StatusChanges {
			if added[change.ApplicationID] {
				change.ID = 0
				changes = append(changes, change)
			}
		}
		var followUps []domain.FollowUp
		for _, followUp := range p.backup.FollowUps {
			if added[followUp.ApplicationID] {
				followUp.ID = 0
				followUps = append(followUps, followUp)
			}
		}
		if err := insert(ctx, tx, changes); err != nil {
			return err
		}
		return insert(ctx, tx, followUps)
	}))
}

func (p *run) fail(index int, err error) {
	p.cache.result.Failed = append(p.cache.result.Failed, domain.BatchFailure{Index: &index, Error: err.Error()})
}

// Helpers

// takenIDs are the ids of the applications and of the trash.
func takenIDs(ctx context.Context, tx bun.Tx) (map[string]bool, error) {
	var ids []string
	err := tx.NewRaw("SELECT id FROM applications UNION SELECT id FROM deleted_applications").Scan(ctx, &ids)
	taken := make(map[string]bool, len(ids))
	for _, id := range ids {
		taken[id] = true
	}
	return taken, err
}

// chunk is the most rows of one insert: SQLite caps the variables of a query, and an application has twenty columns.
const chunk = 200

// insert writes items by chunks, none when there are none (bun refuses an empty insert).
func insert[T any](ctx context.Context, tx bun.Tx, items []T) error {
	for len(items) > 0 {
		n := min(chunk, len(items))
		batch := items[:n]
		if _, err := tx.NewInsert().Model(&batch).Exec(ctx); err != nil {
			return err
		}
		items = items[n:]
	}
	return nil
}
