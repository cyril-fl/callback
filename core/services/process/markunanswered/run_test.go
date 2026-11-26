package markunanswered_test

import (
	"slices"
	"testing"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/createapplications"
	"github.com/cyril-fl/callback/core/services/process/markunanswered"
	"github.com/cyril-fl/callback/core/services/process/updateapplications"
	"github.com/cyril-fl/callback/core/services/testdb"
)

// Only the applications waiting without news for the delay move, the last follow-up restarting it, with a change
// undo_status takes back; 0 moves none.
func TestRun(t *testing.T) {
	db := testdb.New(t)
	ctx := testdb.Account(t, db, "test")
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	create := createapplications.Process(ctx, db, []domain.NewApplication{
		{Company: "Old", Location: "Paris", AppliedAt: "2026-01-01"},
		{Company: "Recent", Location: "Paris", AppliedAt: "2026-02-20"},
		{Company: "Offer", Location: "Paris", AppliedAt: "2026-01-01", Status: domain.StatusOffer},
		{Company: "Followed up", Location: "Paris", AppliedAt: "2026-01-01", Status: domain.StatusResponded},
		{Company: "Interview to come", Location: "Paris", AppliedAt: "2026-01-01", InterviewAt: "2026-03-05T10:00"},
	})
	create.Run()
	if create.HasErr() {
		t.Fatal(create.GetErr())
	}
	ids := create.GetResult().IDs
	// Followed up on 2026-02-25: the delay runs from there, not from the day it applied.
	if _, err := db.NewInsert().Model(&domain.FollowUp{ApplicationID: ids[3], At: time.Date(2026, 2, 25, 9, 0, 0, 0, time.UTC)}).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.NewUpdate().Model((*domain.Application)(nil)).Set("followed_up_at = ?", time.Date(2026, 2, 25, 9, 0, 0, 0, time.UTC)).Where("id = ?", ids[3]).Exec(ctx); err != nil {
		t.Fatal(err)
	}

	off := markunanswered.Process(ctx, db, 0, now)
	off.Run()
	if off.HasErr() || off.GetResult() != 0 {
		t.Fatalf("days 0: %d, %v", off.GetResult(), off.GetErr())
	}
	mark := markunanswered.Process(ctx, db, 10, now)
	mark.Run()
	if mark.HasErr() || mark.GetResult() != 1 {
		t.Fatalf("marked %d, %v; want 1", mark.GetResult(), mark.GetErr())
	}
	var noReply []string
	if err := db.NewSelect().Model((*domain.Application)(nil)).Column("id").Where("status = ?", domain.StatusNoReply).Scan(ctx, &noReply); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(noReply, ids[:1]) {
		t.Fatalf("no_reply = %v, want %v", noReply, ids[:1])
	}
	// The change is news: running again moves nothing more.
	again := markunanswered.Process(ctx, db, 10, now)
	again.Run()
	if again.GetResult() != 0 {
		t.Fatalf("again: %d", again.GetResult())
	}

	undo := updateapplications.Process(ctx, db, updateapplications.Input{Ops: []updateapplications.Op{{Filter: domain.Filter{IDs: ids[:1]}, Set: domain.Changes{UndoStatus: true}}}})
	undo.Run()
	if undo.HasErr() || undo.GetResult().Count != 1 {
		t.Fatalf("undo: %+v, %v", undo.GetResult(), undo.GetErr())
	}
}
