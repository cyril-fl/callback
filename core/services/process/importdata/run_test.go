package importdata_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/createapplications"
	"github.com/cyril-fl/callback/core/services/process/deleteapplications"
	"github.com/cyril-fl/callback/core/services/process/exportdata"
	"github.com/cyril-fl/callback/core/services/process/importdata"
	"github.com/cyril-fl/callback/core/services/process/updateapplications"
	"github.com/cyril-fl/callback/core/services/testdb"
)

// An export restores into an empty base with its history and trash; importing it again changes nothing, and an
// invalid application is listed without stopping the others.
func TestRoundTrip(t *testing.T) {
	from := testdb.New(t)
	ctx := testdb.Account(t, from, "test")
	create := createapplications.Process(ctx, from, []domain.NewApplication{{Company: "Acme", Location: "Paris"}, {Company: "Globex", Location: "Lille"}})
	create.Run()
	ids := create.GetResult().IDs
	status := domain.StatusInterview
	update := updateapplications.Process(ctx, from, updateapplications.Input{Ops: []updateapplications.Op{{Filter: domain.Filter{IDs: ids[:1]}, Set: domain.Changes{Status: &status, FollowedUp: true}}}})
	update.Run()
	remove := deleteapplications.Process(ctx, from, deleteapplications.Input{Filter: domain.Filter{IDs: ids[1:]}})
	remove.Run()
	for _, p := range []interface{ GetErr() error }{create, update, remove} {
		if p.GetErr() != nil {
			t.Fatal(p.GetErr())
		}
	}

	export := exportdata.Process(ctx, from)
	export.Run()
	if export.HasErr() {
		t.Fatal(export.GetErr())
	}
	// Through JSON, as the API does.
	data, err := json.Marshal(export.GetResult())
	if err != nil {
		t.Fatal(err)
	}
	var backup domain.Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		t.Fatal(err)
	}
	backup.Applications = append(backup.Applications, domain.Application{ID: "bad", Location: "Nice"})
	// The trash's restore inserts the application as it is: the item's id wins over the one inside.
	backup.Deleted[0].Application.ID = "other"

	to := testdb.New(t)
	for pass := 1; pass <= 2; pass++ {
		restore := importdata.Process(ctx, to, backup)
		restore.Run()
		if restore.HasErr() {
			t.Fatal(restore.GetErr())
		}
		got := restore.GetResult()
		want := importdata.Result{Added: 1, Deleted: 1}
		if pass == 2 {
			want = importdata.Result{Skipped: 2}
		}
		if got.Added != want.Added || got.Deleted != want.Deleted || got.Skipped != want.Skipped || len(got.Failed) != 1 || *got.Failed[0].Index != 1 {
			t.Fatalf("pass %d: %+v", pass, got)
		}
	}

	again := exportdata.Process(ctx, to)
	again.Run()
	got := again.GetResult()
	if got.Deleted[0].Application.ID != got.Deleted[0].ID {
		t.Fatalf("deleted %q holds application %q", got.Deleted[0].ID, got.Deleted[0].Application.ID)
	}
	if len(got.Applications) != 1 || got.Applications[0].Company != "Acme" || len(got.Deleted) != 1 || len(got.StatusChanges) != 3 || len(got.FollowUps) != 1 {
		t.Fatalf("restored: %d applications, %d deleted, %d changes, %d follow-ups", len(got.Applications), len(got.Deleted), len(got.StatusChanges), len(got.FollowUps))
	}
}

func TestWrongVersion(t *testing.T) {
	restore := importdata.Process(context.Background(), testdb.New(t), domain.Backup{Version: 99})
	restore.Run()
	if !errors.Is(restore.GetErr(), domain.ErrInvalidBackup) {
		t.Fatalf("err = %v", restore.GetErr())
	}
}
