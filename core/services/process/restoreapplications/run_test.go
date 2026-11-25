package restoreapplications_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/deleteapplications"
	"github.com/cyril-fl/callback/core/services/process/listdeletedapplications"
	"github.com/cyril-fl/callback/core/services/process/restoreapplications"
	"github.com/cyril-fl/callback/core/services/seed"
	"github.com/cyril-fl/callback/core/services/testdb"
)

func TestRun(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	ids := seed.Applications(t, db,
		domain.NewApplication{Company: "Acme", Location: "Paris", Status: domain.StatusInterview, Rating: 4},
		domain.NewApplication{Company: "Globex", Location: "Lyon"},
	)
	var before domain.Application
	if err := db.NewSelect().Model(&before).Where("id = ?", ids[0]).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	remove := deleteapplications.Process(ctx, db, deleteapplications.Input{Filter: domain.Filter{IDs: ids}})
	remove.Run()
	if remove.HasErr() || remove.GetResult().Count != 2 {
		t.Fatalf("delete: %v %+v", remove.GetErr(), remove.GetResult())
	}
	if count, _ := db.NewSelect().Model((*domain.Application)(nil)).Count(ctx); count != 0 {
		t.Fatalf("%d applications left after the delete", count)
	}

	list := listdeletedapplications.Process(ctx, db)
	if list.Run(); list.HasErr() || len(list.GetResult()) != 2 || list.GetResult()[0].Application.Company == "" {
		t.Fatalf("trash = %v %+v", list.GetErr(), list.GetResult())
	}

	missing := "00000000-0000-0000-0000-000000000000"
	process := restoreapplications.Process(ctx, db, restoreapplications.Input{IDs: []string{ids[0], missing}})
	process.Run()
	if process.HasErr() {
		t.Fatal(process.GetErr())
	}
	if got := process.GetResult(); got.Count != 1 || len(got.Failed) != 1 || got.Failed[0].ID != missing {
		t.Fatalf("restore = %+v", got)
	}
	var after domain.Application
	if err := db.NewSelect().Model(&after).Where("id = ?", ids[0]).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if after.Company != before.Company || after.Status != before.Status || after.Rating != before.Rating || !after.CreatedAt.Equal(before.CreatedAt) {
		t.Errorf("restored %+v, want %+v", after, before)
	}
	if count, _ := db.NewSelect().Model((*domain.StatusChange)(nil)).Where("application_id = ?", ids[0]).Count(ctx); count != 1 {
		t.Errorf("the restored application has %d status changes, want its creation's", count)
	}

	// trash_days later the trash is emptied, the history with it, and nothing comes back.
	if _, err := db.NewUpdate().Model((*domain.DeletedApplication)(nil)).Set("deleted_at = ?", time.Now().UTC().AddDate(0, 0, -domain.DefaultPreferences.TrashDays).Add(-time.Minute)).Where("1 = 1").Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if list.Run(); len(list.GetResult()) != 0 {
		t.Errorf("the trash lists %d expired applications", len(list.GetResult()))
	}
	process = restoreapplications.Process(ctx, db, restoreapplications.Input{IDs: []string{ids[1]}})
	process.Run()
	if got := process.GetResult(); process.HasErr() || got.Count != 0 || len(got.Failed) != 1 {
		t.Errorf("restore of an expired one = %v %+v", process.GetErr(), got)
	}
	if count, _ := db.NewSelect().Model((*domain.StatusChange)(nil)).Where("application_id = ?", ids[1]).Count(ctx); count != 0 {
		t.Errorf("the expired application kept %d status changes", count)
	}

	empty := restoreapplications.Process(ctx, db, restoreapplications.Input{})
	if empty.Run(); !errors.Is(empty.GetErr(), domain.ErrEmptyBatch) {
		t.Errorf("empty restore = %v", empty.GetErr())
	}
}
