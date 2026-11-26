package listapplications_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/listapplications"
	"github.com/cyril-fl/callback/core/services/seed"
	"github.com/cyril-fl/callback/core/services/testdb"
)

func TestRun(t *testing.T) {
	db := testdb.New(t)
	ctx := testdb.Account(t, db, "test")
	ids := seed.Applications(t, ctx, db,
		domain.NewApplication{Company: "Acme", Location: "Paris", AppliedAt: "2026-01-10", Comment: "Called back 100% sure"},
		domain.NewApplication{Company: "Globex", Location: "Lyon", AppliedAt: "2026-02-10", Status: domain.StatusInterview, Rating: 4, ContactName: "Jane Doe", Title: " Backend Developer "},
		domain.NewApplication{Company: "Initech", Location: "Paris", AppliedAt: "2026-03-10", Status: domain.StatusRejected},
	)
	acme, globex, initech := ids[0], ids[1], ids[2]

	tests := map[string]struct {
		input listapplications.Input
		want  []string
		total int
	}{
		"all, newest first":   {listapplications.Input{}, []string{initech, globex, acme}, 3},
		"by status":           {listapplications.Input{Filter: domain.Filter{Status: []domain.Status{domain.StatusInterview, domain.StatusRejected}}}, []string{initech, globex}, 2},
		"location, any case":  {listapplications.Input{Filter: domain.Filter{Location: "PARIS"}}, []string{initech, acme}, 2},
		"text, any case":      {listapplications.Input{Filter: domain.Filter{Q: "paris"}}, []string{initech, acme}, 2},
		"text, % is literal":  {listapplications.Input{Filter: domain.Filter{Q: "100%"}}, []string{acme}, 1},
		"text, contact":       {listapplications.Input{Filter: domain.Filter{Q: "jane"}}, []string{globex}, 1},
		"text, title":         {listapplications.Input{Filter: domain.Filter{Q: "backend"}}, []string{globex}, 1},
		"text, % alone":       {listapplications.Input{Filter: domain.Filter{Q: "%%"}}, nil, 0},
		"dates are strict":    {listapplications.Input{Filter: domain.Filter{AppliedAfter: "2026-01-10", AppliedBefore: "2026-03-10"}}, []string{globex}, 1},
		"ids":                 {listapplications.Input{Filter: domain.Filter{IDs: []string{acme, initech}}}, []string{initech, acme}, 2},
		"sort by company":     {listapplications.Input{Sort: "company"}, []string{acme, globex, initech}, 3},
		"sort by status":      {listapplications.Input{Sort: "status"}, []string{acme, globex, initech}, 3},
		"sort by rating desc": {listapplications.Input{Sort: "-rating", Limit: 1}, []string{globex}, 3},
		"page":                {listapplications.Input{Limit: 1, Offset: 1}, []string{globex}, 3},
		"offset alone":        {listapplications.Input{Offset: 2}, []string{acme}, 3},
		"count only":          {listapplications.Input{CountOnly: true}, nil, 3},
		"untouched since":     {listapplications.Input{Filter: domain.Filter{UntouchedSince: "2000-01-01"}}, nil, 0},
		"untouched, all":      {listapplications.Input{Filter: domain.Filter{UntouchedSince: "2100-01-01"}}, []string{initech, globex, acme}, 3},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			process := listapplications.Process(ctx, db, test.input)
			process.Run()
			if process.HasErr() {
				t.Fatal(process.GetErr())
			}
			var got []string
			for _, application := range process.GetResult().Applications {
				got = append(got, application.ID)
			}
			if !slices.Equal(got, test.want) || process.GetResult().Total != test.total {
				t.Errorf("got %v (total %d), want %v (total %d)", got, process.GetResult().Total, test.want, test.total)
			}
		})
	}
}

// fields keeps the columns asked and id; max_chars cuts the comment and says so.
func TestFieldsAndMaxChars(t *testing.T) {
	db := testdb.New(t)
	ctx := testdb.Account(t, db, "test")
	seed.Applications(t, ctx, db, domain.NewApplication{Company: "Acme", Location: "Paris", Comment: "éèàùç and more"})

	process := listapplications.Process(ctx, db, listapplications.Input{Fields: []string{"company", "comment"}, MaxChars: 5})
	process.Run()
	if process.HasErr() {
		t.Fatal(process.GetErr())
	}
	got := process.GetResult().Applications[0]
	if got.ID == "" || got.Company != "Acme" || got.Location != "" || got.Comment != "éèàùç" || !got.Truncated {
		t.Errorf("got %+v", got)
	}
}

func TestInvalid(t *testing.T) {
	db := testdb.New(t)
	ctx := testdb.Account(t, db, "test")
	for name, input := range map[string]listapplications.Input{
		"field":  {Fields: []string{"password"}},
		"sort":   {Sort: "id; DROP TABLE applications"},
		"limit":  {Limit: -1},
		"status": {Filter: domain.Filter{Status: []domain.Status{"hired"}}},
		"date":   {Filter: domain.Filter{AppliedBefore: "yesterday"}},
	} {
		process := listapplications.Process(ctx, db, input)
		process.Run()
		if !errors.Is(process.GetErr(), domain.ErrInvalid) {
			t.Errorf("%s: err = %v, want invalid", name, process.GetErr())
		}
	}
}

// status sorts in the pipeline's order: responded comes before interview, though not in the alphabet.
func TestSortStatus(t *testing.T) {
	db := testdb.New(t)
	ctx := testdb.Account(t, db, "test")
	ids := seed.Applications(t, ctx, db,
		domain.NewApplication{Company: "A", Location: "Paris", Status: domain.StatusInterview},
		domain.NewApplication{Company: "B", Location: "Paris", Status: domain.StatusResponded},
	)
	process := listapplications.Process(ctx, db, listapplications.Input{Sort: "status"})
	process.Run()
	if got := process.GetResult().Applications; process.HasErr() || len(got) != 2 || got[0].ID != ids[1] {
		t.Errorf("got %+v, %v; want responded first", got, process.GetErr())
	}
}

// position sorts the kanban: a placed card by its place, a new one (no position) on top, newest first.
func TestSortPosition(t *testing.T) {
	db := testdb.New(t)
	ctx := testdb.Account(t, db, "test")
	ids := seed.Applications(t, ctx, db,
		domain.NewApplication{Company: "A", Location: "Paris"},
		domain.NewApplication{Company: "B", Location: "Paris"},
		domain.NewApplication{Company: "C", Location: "Paris"},
	)
	for i, id := range []string{ids[2], ids[0]} {
		if _, err := db.NewUpdate().Model((*domain.Application)(nil)).Set("position = ?", i+1).Where("id = ?", id).Exec(ctx); err != nil {
			t.Fatal(err)
		}
	}
	process := listapplications.Process(ctx, db, listapplications.Input{Sort: "position"})
	process.Run()
	var got []string
	for _, application := range process.GetResult().Applications {
		got = append(got, application.ID)
	}
	if want := []string{ids[1], ids[2], ids[0]}; process.HasErr() || !slices.Equal(got, want) {
		t.Errorf("got %v, %v; want %v", got, process.GetErr(), want)
	}
}
