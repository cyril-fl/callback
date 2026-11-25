package updateapplications_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cyril-fl/jat/core/domain"
	"github.com/cyril-fl/jat/core/services/process/updateapplications"
	"github.com/cyril-fl/jat/core/services/seed"
	"github.com/cyril-fl/jat/core/services/testdb"
)

func ptr[T any](value T) *T { return &value }

func TestRun(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	ids := seed.Applications(t, db,
		domain.NewApplication{Company: "Acme", Location: "Paris"},
		domain.NewApplication{Company: "Globex", Location: "Lyon"},
		domain.NewApplication{Company: "Initech", Location: "Lille", Status: domain.StatusRejected},
	)
	missing := "00000000-0000-0000-0000-000000000000"
	input := updateapplications.Input{Ops: []updateapplications.Op{
		{Filter: domain.Filter{Status: []domain.Status{domain.StatusApplied}}, Set: domain.Changes{Status: ptr(domain.StatusResponded)}},
		{Filter: domain.Filter{IDs: []string{ids[2], missing}}, Set: domain.Changes{Rating: ptr(9)}},
		{Filter: domain.Filter{}, Set: domain.Changes{Rating: ptr(1)}},
		{Filter: domain.Filter{IDs: []string{ids[0]}}},
	}}

	// The dry run says what the call writes, and writes nothing.
	dry := input
	dry.DryRun, dry.Summary = true, true
	process := updateapplications.Process(ctx, db, dry)
	process.Run()
	if process.HasErr() {
		t.Fatal(process.GetErr())
	}
	if got := process.GetResult(); got.Count != 2 || got.IDs != nil || !got.DryRun || len(got.Failed) != 4 {
		t.Fatalf("dry run = %+v", got)
	}
	if count, _ := db.NewSelect().Model((*domain.Application)(nil)).Where("status = ?", domain.StatusResponded).Count(ctx); count != 0 {
		t.Fatalf("the dry run wrote %d applications", count)
	}

	process = updateapplications.Process(ctx, db, input)
	process.Run()
	if process.HasErr() {
		t.Fatal(process.GetErr())
	}
	result := process.GetResult()
	if result.Count != 2 || len(result.IDs) != 2 {
		t.Errorf("result = %+v, want the 2 applied ones", result)
	}
	wants := []struct {
		op  int
		id  string
		err error
	}{
		{1, missing, domain.ErrNotFound},
		{1, ids[2], domain.ErrInvalidRating},
		{2, "", domain.ErrNoFilter},
		{3, "", domain.ErrNoChanges},
	}
	if len(result.Failed) != len(wants) {
		t.Fatalf("failed = %+v", result.Failed)
	}
	for i, want := range wants {
		got := result.Failed[i]
		if *got.Op != want.op || got.ID != want.id || (want.err != domain.ErrNotFound && got.Error != want.err.Error()) {
			t.Errorf("failure %d = %+v, want %+v", i, got, want)
		}
	}
	var acme domain.Application
	if err := db.NewSelect().Model(&acme).Where("id = ?", ids[0]).Scan(ctx); err != nil || acme.Status != domain.StatusResponded || !acme.UpdatedAt.After(acme.CreatedAt) && !acme.UpdatedAt.Equal(acme.CreatedAt) {
		t.Errorf("acme = %+v, %v", acme, err)
	}
}

func TestNoOps(t *testing.T) {
	process := updateapplications.Process(context.Background(), testdb.New(t), updateapplications.Input{})
	process.Run()
	if !errors.Is(process.GetErr(), domain.ErrInvalid) {
		t.Errorf("err = %v, want invalid", process.GetErr())
	}
}

// A q of spaces adds no condition: the op fails instead of changing every application.
func TestBlankQuery(t *testing.T) {
	db := testdb.New(t)
	seed.Applications(t, db, domain.NewApplication{Company: "Acme", Location: "Paris"})
	process := updateapplications.Process(context.Background(), db, updateapplications.Input{Ops: []updateapplications.Op{{Filter: domain.Filter{Q: " "}, Set: domain.Changes{Rating: ptr(1)}}}})
	process.Run()
	if got := process.GetResult(); process.HasErr() || got.Count != 0 || len(got.Failed) != 1 || got.Failed[0].Error != domain.ErrNoFilter.Error() {
		t.Errorf("result = %+v, %v; want the op refused", got, process.GetErr())
	}
}
