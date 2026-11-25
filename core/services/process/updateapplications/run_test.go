package updateapplications_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/appfilter"
	"github.com/cyril-fl/callback/core/services/process/updateapplications"
	"github.com/cyril-fl/callback/core/services/seed"
	"github.com/cyril-fl/callback/core/services/testdb"
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

// A follow-up is news: the application leaves untouched_since until the delay passes again.
func TestFollowUp(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	ids := seed.Applications(t, db,
		domain.NewApplication{Company: "Acme", Location: "Paris"},
		domain.NewApplication{Company: "Globex", Location: "Lyon"},
	)
	// Both reached their status long ago.
	if _, err := db.NewUpdate().Model((*domain.StatusChange)(nil)).Set("at = ?", time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)).Where("1 = 1").Exec(ctx); err != nil {
		t.Fatal(err)
	}
	process := updateapplications.Process(ctx, db, updateapplications.Input{Ops: []updateapplications.Op{
		{Filter: domain.Filter{IDs: []string{ids[0]}}, Set: domain.Changes{FollowedUp: true}},
	}})
	process.Run()
	if process.HasErr() || process.GetResult().Count != 1 {
		t.Fatal(process.GetErr(), process.GetResult())
	}

	var quiet []domain.Application
	since := domain.Filter{UntouchedSince: time.Now().UTC().Format(domain.DateLayout)}
	if err := db.NewSelect().Model(&quiet).ApplyQueryBuilder(appfilter.Where(since)).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(quiet) != 1 || quiet[0].ID != ids[1] {
		t.Errorf("untouched since today = %+v, want Globex only", quiet)
	}
	var acme domain.Application
	if err := db.NewSelect().Model(&acme).Where("id = ?", ids[0]).Scan(ctx); err != nil || acme.FollowedUpAt.IsZero() {
		t.Errorf("acme = %+v, %v", acme, err)
	}
	if count, _ := db.NewSelect().Model((*domain.FollowUp)(nil)).Where("application_id = ?", ids[0]).Count(ctx); count != 1 {
		t.Errorf("%d follow-ups, want 1", count)
	}
}

// The filter on source ignores case, a set source reuses the name in use, a status change records the status before.
func TestSourceCaseAndFrom(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	ids := seed.Applications(t, db,
		domain.NewApplication{Company: "A", Location: "Paris", Source: "LinkedIn"},
		domain.NewApplication{Company: "B", Location: "Paris", Source: "Indeed"},
	)
	process := updateapplications.Process(ctx, db, updateapplications.Input{Ops: []updateapplications.Op{
		{Filter: domain.Filter{Source: "LINKEDIN"}, Set: domain.Changes{Status: ptr(domain.StatusRejected)}},
		{Filter: domain.Filter{IDs: []string{ids[1]}}, Set: domain.Changes{Source: ptr(domain.Source("linkedin"))}},
	}})
	process.Run()
	if result := process.GetResult(); process.HasErr() || result.Count != 2 || len(result.Failed) != 0 {
		t.Fatalf("result = %+v, %v", result, process.GetErr())
	}
	var b domain.Application
	if err := db.NewSelect().Model(&b).Where("id = ?", ids[1]).Scan(ctx); err != nil || b.Source != "LinkedIn" {
		t.Errorf("source = %q, %v, want LinkedIn", b.Source, err)
	}
	var change domain.StatusChange
	if err := db.NewSelect().Model(&change).Where("application_id = ? AND status = ?", ids[0], domain.StatusRejected).Scan(ctx); err != nil || change.From != domain.StatusApplied {
		t.Errorf("change = %+v, %v, want from applied", change, err)
	}
}

func TestUndo(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	ids := seed.Applications(t, db, domain.NewApplication{Company: "Acme", Location: "Paris"})
	update := func(set domain.Changes) domain.BatchResult {
		t.Helper()
		process := updateapplications.Process(ctx, db, updateapplications.Input{Ops: []updateapplications.Op{{Filter: domain.Filter{IDs: ids}, Set: set}}})
		process.Run()
		if process.HasErr() {
			t.Fatal(process.GetErr())
		}
		return process.GetResult()
	}
	read := func() (domain.Application, int64, int64) {
		t.Helper()
		var application domain.Application
		if err := db.NewSelect().Model(&application).Where("id = ?", ids[0]).Scan(ctx); err != nil {
			t.Fatal(err)
		}
		changes, _ := db.NewSelect().Model((*domain.StatusChange)(nil)).Where("application_id = ?", ids[0]).Count(ctx)
		followUps, _ := db.NewSelect().Model((*domain.FollowUp)(nil)).Where("application_id = ?", ids[0]).Count(ctx)
		return application, changes, followUps
	}

	// Nothing to undo yet: the creation is not a change, and no follow-up was made.
	if got := update(domain.Changes{UndoStatus: true}); got.Count != 0 || len(got.Failed) != 1 || got.Failed[0].Error != domain.ErrNoStatusChange.Error() {
		t.Fatalf("undo of the creation = %+v", got)
	}
	if got := update(domain.Changes{UndoFollowUp: true}); got.Count != 0 || len(got.Failed) != 1 {
		t.Fatalf("undo without follow-up = %+v", got)
	}
	if got := update(domain.Changes{UndoStatus: true, Status: ptr(domain.StatusOffer)}); len(got.Failed) != 1 || got.Failed[0].Error != domain.ErrUndoConflict.Error() {
		t.Fatalf("undo with a status = %+v", got)
	}

	update(domain.Changes{Status: ptr(domain.StatusInterview)})
	update(domain.Changes{FollowedUp: true})
	first, _, _ := read()
	update(domain.Changes{FollowedUp: true})
	update(domain.Changes{UndoStatus: true, UndoFollowUp: true})
	application, changes, followUps := read()
	if application.Status != domain.StatusApplied || changes != 1 {
		t.Errorf("after the status undo: %s with %d changes, want applied with its creation only", application.Status, changes)
	}
	if !application.FollowedUpAt.Equal(first.FollowedUpAt) || followUps != 1 {
		t.Errorf("after the follow-up undo: %v with %d follow-ups, want the first one %v", application.FollowedUpAt, followUps, first.FollowedUpAt)
	}
	update(domain.Changes{UndoFollowUp: true})
	if application, _, followUps = read(); !application.FollowedUpAt.IsZero() || followUps != 0 {
		t.Errorf("after the last follow-up undo: %v with %d follow-ups, want none", application.FollowedUpAt, followUps)
	}
}
