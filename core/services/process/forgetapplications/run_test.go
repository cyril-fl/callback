package forgetapplications_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/deleteapplications"
	"github.com/cyril-fl/callback/core/services/process/forgetapplications"
	"github.com/cyril-fl/callback/core/services/seed"
	"github.com/cyril-fl/callback/core/services/testdb"
)

func TestRun(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	ids := seed.Applications(t, db,
		domain.NewApplication{Company: "Acme", Location: "Paris"},
		domain.NewApplication{Company: "Globex", Location: "Lyon"},
		domain.NewApplication{Company: "Initech", Location: "Lille"},
	)
	remove := deleteapplications.Process(ctx, db, deleteapplications.Input{Filter: domain.Filter{IDs: ids[:2]}})
	if remove.Run(); remove.HasErr() {
		t.Fatal(remove.GetErr())
	}
	forget := func(input forgetapplications.Input) (domain.BatchResult, error) {
		process := forgetapplications.Process(ctx, db, input)
		process.Run()
		return process.GetResult(), process.GetErr()
	}
	count := func(model any) int64 {
		n, _ := db.NewSelect().Model(model).Count(ctx)
		return n
	}

	for _, input := range []forgetapplications.Input{{}, {IDs: ids[:1], All: true}} {
		if _, err := forget(input); !errors.Is(err, forgetapplications.ErrNoIDs) {
			t.Errorf("%+v = %v, want ErrNoIDs", input, err)
		}
	}

	// One by id: the application left out of the trash is not touched.
	got, err := forget(forgetapplications.Input{IDs: []string{ids[0], ids[2]}})
	if err != nil || got.Count != 1 || len(got.Failed) != 1 || got.Failed[0].ID != ids[2] {
		t.Fatalf("forget = %v %+v", err, got)
	}
	if n := count((*domain.DeletedApplication)(nil)); n != 1 {
		t.Errorf("%d in the trash, want 1", n)
	}

	// All: the trash is empty, its history gone, the applications kept.
	if got, err = forget(forgetapplications.Input{All: true}); err != nil || got.Count != 1 {
		t.Fatalf("empty = %v %+v", err, got)
	}
	if n := count((*domain.DeletedApplication)(nil)); n != 0 {
		t.Errorf("%d left in the trash", n)
	}
	if n := count((*domain.StatusChange)(nil)); n != 1 {
		t.Errorf("%d status changes, want the kept application's", n)
	}
	if n := count((*domain.Application)(nil)); n != 1 {
		t.Errorf("%d applications, want 1", n)
	}
}
