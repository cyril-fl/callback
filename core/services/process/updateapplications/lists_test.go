package updateapplications_test

import (
	"testing"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/applists"
	"github.com/cyril-fl/callback/core/services/process/updateapplications"
	"github.com/cyril-fl/callback/core/services/seed"
	"github.com/cyril-fl/callback/core/services/testdb"
)

// The lists are written whole, and interview_at and the contact follow them; without a list, those columns edit
// the last interview and the first contact.
func TestLists(t *testing.T) {
	db := testdb.New(t)
	ctx := testdb.Account(t, db, "test")
	ids := seed.Applications(t, ctx, db, domain.NewApplication{Company: "Acme", Location: "Paris", InterviewAt: "2026-02-01T10:00", ContactName: "Léa"})

	read := func() domain.Application {
		t.Helper()
		var applications []domain.Application
		if err := db.NewSelect().Model(&applications).Where("id = ?", ids[0]).Scan(ctx); err != nil {
			t.Fatal(err)
		}
		if err := applists.Load(ctx, db, applications); err != nil {
			t.Fatal(err)
		}
		return applications[0]
	}
	update := func(set domain.Changes) domain.BatchResult {
		t.Helper()
		process := updateapplications.Process(ctx, db, updateapplications.Input{Ops: []updateapplications.Op{{Filter: domain.Filter{IDs: ids}, Set: set}}})
		if process.Run(); process.HasErr() {
			t.Fatal(process.GetErr())
		}
		return process.GetResult()
	}

	if a := read(); len(a.Interviews) != 1 || a.Interviews[0].Kind != domain.InterviewOther || len(a.Contacts) != 1 || a.Contacts[0].Name != "Léa" {
		t.Fatalf("created = %+v %+v, want one of each from the columns", a.Interviews, a.Contacts)
	}

	update(domain.Changes{
		Interviews: &[]domain.Interview{{At: "2026-03-01T09:00", Kind: domain.InterviewTech, Note: " live coding "}, {At: "2026-02-01T10:00", Kind: domain.InterviewHR}},
		Contacts:   &[]domain.Contact{{Name: "Max", Role: "CTO"}, {Email: "jobs@acme.fr"}},
	})
	a := read()
	if len(a.Interviews) != 2 || a.Interviews[1].Note != "live coding" || a.InterviewAt != "2026-03-01T09:00" || a.ContactName != "Max" || len(a.Contacts) != 2 {
		t.Fatalf("lists = %+v %+v, interview_at %q, contact %q", a.Interviews, a.Contacts, a.InterviewAt, a.ContactName)
	}

	update(domain.Changes{InterviewAt: ptr(""), ContactName: ptr(""), Rating: ptr(3)})
	a = read()
	if len(a.Interviews) != 1 || a.InterviewAt != "2026-02-01T10:00" || len(a.Contacts) != 1 || a.ContactEmail != "jobs@acme.fr" {
		t.Fatalf("after the column edits = %+v %+v", a.Interviews, a.Contacts)
	}

	update(domain.Changes{Rating: ptr(4)})
	if a := read(); len(a.Interviews) != 1 || len(a.Contacts) != 1 {
		t.Fatalf("a change without lists touched them: %+v %+v", a.Interviews, a.Contacts)
	}

	if result := update(domain.Changes{Interviews: &[]domain.Interview{{At: "tomorrow"}}, Contacts: &[]domain.Contact{{Role: "HR"}}}); len(result.Failed) != 1 || result.Count != 0 {
		t.Fatalf("invalid lists = %+v", result)
	}
}
