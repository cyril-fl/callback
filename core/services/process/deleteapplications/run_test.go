package deleteapplications_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/deleteapplications"
	"github.com/cyril-fl/callback/core/services/seed"
	"github.com/cyril-fl/callback/core/services/testdb"
)

func TestRun(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	seed.Applications(t, db,
		domain.NewApplication{Company: "Acme", Location: "Paris", Status: domain.StatusRejected},
		domain.NewApplication{Company: "Globex", Location: "Lyon", Status: domain.StatusRejected},
		domain.NewApplication{Company: "Initech", Location: "Lille"},
	)
	count := func() int {
		count, err := db.NewSelect().Model((*domain.Application)(nil)).Count(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return int(count)
	}

	// An empty filter would delete everything: refused.
	empty := deleteapplications.Process(ctx, db, deleteapplications.Input{})
	empty.Run()
	if !errors.Is(empty.GetErr(), domain.ErrNoFilter) || count() != 3 {
		t.Fatalf("empty filter: err = %v, count = %d", empty.GetErr(), count())
	}

	rejected := domain.Filter{Status: []domain.Status{domain.StatusRejected}}
	dry := deleteapplications.Process(ctx, db, deleteapplications.Input{Filter: rejected, DryRun: true, Summary: true})
	dry.Run()
	if got := dry.GetResult(); dry.HasErr() || got.Count != 2 || got.IDs != nil || !got.DryRun || count() != 3 {
		t.Fatalf("dry run = %+v, %v, count = %d", got, dry.GetErr(), count())
	}

	missing := "00000000-0000-0000-0000-000000000000"
	process := deleteapplications.Process(ctx, db, deleteapplications.Input{Filter: domain.Filter{Status: rejected.Status, IDs: nil}})
	process.Run()
	if got := process.GetResult(); process.HasErr() || got.Count != 2 || len(got.IDs) != 2 || count() != 1 {
		t.Fatalf("delete = %+v, %v, count = %d", got, process.GetErr(), count())
	}

	gone := deleteapplications.Process(ctx, db, deleteapplications.Input{Filter: domain.Filter{IDs: []string{missing}}})
	gone.Run()
	if got := gone.GetResult(); gone.HasErr() || got.Count != 0 || len(got.Failed) != 1 || got.Failed[0].ID != missing {
		t.Errorf("missing id = %+v, %v", got, gone.GetErr())
	}
}

// A q of spaces adds no condition: it would match everything, so it is refused like an empty filter.
func TestBlankQuery(t *testing.T) {
	db := testdb.New(t)
	seed.Applications(t, db, domain.NewApplication{Company: "Acme", Location: "Paris"})
	process := deleteapplications.Process(context.Background(), db, deleteapplications.Input{Filter: domain.Filter{Q: "  "}})
	process.Run()
	if !errors.Is(process.GetErr(), domain.ErrNoFilter) {
		t.Errorf("err = %v, want %v", process.GetErr(), domain.ErrNoFilter)
	}
}
