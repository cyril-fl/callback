package createapplications_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/createapplications"
	"github.com/cyril-fl/callback/core/services/testdb"
)

// The valid items of a batch are written, the invalid ones listed by index; the defaults fill what is left out.
func TestRun(t *testing.T) {
	db := testdb.New(t)
	ctx := testdb.Account(t, db, "test")
	items := []domain.NewApplication{
		{Company: " Acme ", Location: "Paris"},
		{Company: "", Location: "Lyon"},
		{Company: "Globex", Location: "Lille", Source: domain.SourceJobPost, SourceDetail: "https://jobs.example/1"},
		{Company: "Initech", Location: "Nantes", Source: domain.SourceRecruiter, SourceDetail: "not an e-mail"},
		{Company: "Umbrella", Location: "Brest", Rating: 6},
		{Company: "Hooli", Location: "Rennes", Source: domain.SourceRecruiter, SourceDetail: "jane@hooli.example", Comment: "<b>raw</b>"},
		{Company: "Vandelay", Location: "Lyon", Street: " 1 rue de la Paix ", PostalCode: "69001", Country: "France", Source: " Agence intérim Adecco ", SourceDetail: "Marie, 06 00 00 00 00", InterviewAt: "2026-02-03T14:30", ContactName: " Marie ", ContactEmail: "marie@adecco.example"},
		{Company: "Pied Piper", Location: "Paris", Source: domain.Source(strings.Repeat("x", domain.MaxSource+1))},
		{Company: "Dunder", Location: "Scranton", Source: domain.SourceSpontaneous, SourceDetail: "not for spontaneous"},
		{Company: "Wonka", Location: "Paris", Website: "jobs@wonka.example"},
		{Company: "Krusty", Location: "Paris", Website: "not a site"},
		{Company: "Soylent", Location: "Paris", InterviewAt: "2026-02-03 14:30"},
		{Company: "Tyrell", Location: "Paris", ContactEmail: "Rachel <rachel@tyrell.example>"},
	}
	process := createapplications.Process(ctx, db, items)
	process.Run()
	if process.HasErr() {
		t.Fatal(process.GetErr())
	}
	result := process.GetResult()
	if result.Count != 5 || len(result.IDs) != 5 {
		t.Fatalf("count = %d, ids = %v; want 5", result.Count, result.IDs)
	}
	wants := map[int]error{1: domain.ErrNoCompany, 3: domain.ErrInvalidDetail, 4: domain.ErrInvalidRating, 7: domain.ErrInvalidSource, 8: domain.ErrInvalidDetail, 10: domain.ErrInvalidWebsite, 11: domain.ErrInvalidInterview, 12: domain.ErrInvalidContact}
	if len(result.Failed) != len(wants) {
		t.Fatalf("failed = %+v", result.Failed)
	}
	for _, failure := range result.Failed {
		if want := wants[*failure.Index]; want == nil || failure.Error != want.Error() {
			t.Errorf("failure %+v, want %v", failure, want)
		}
	}

	var acme domain.Application
	if err := db.NewSelect().Model(&acme).Where("id = ?", result.IDs[0]).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if acme.Company != "Acme" || acme.Status != domain.StatusApplied || acme.Source != domain.SourceSpontaneous || !domain.IsDate(acme.AppliedAt) {
		t.Errorf("acme = %+v, want trimmed with the defaults", acme)
	}
	// A source the user names is kept as given, trimmed, its detail free; the address is trimmed too.
	var vandelay domain.Application
	if err := db.NewSelect().Model(&vandelay).Where("id = ?", result.IDs[3]).Scan(ctx); err != nil || vandelay.Source != "Agence intérim Adecco" || vandelay.Street != "1 rue de la Paix" || vandelay.Country != "FR" || vandelay.InterviewAt != "2026-02-03T14:30" || vandelay.ContactName != "Marie" {
		t.Errorf("vandelay = %+v, %v", vandelay, err)
	}
	var hooli domain.Application
	if err := db.NewSelect().Model(&hooli).Where("id = ?", result.IDs[2]).Scan(ctx); err != nil || hooli.Comment != "<b>raw</b>" {
		t.Errorf("comment = %q, %v; want it stored raw", hooli.Comment, err)
	}
}

func TestLimits(t *testing.T) {
	db := testdb.New(t)
	ctx := testdb.Account(t, db, "test")
	for _, items := range [][]domain.NewApplication{nil, make([]domain.NewApplication, domain.MaxBatch+1)} {
		process := createapplications.Process(ctx, db, items)
		process.Run()
		if !errors.Is(process.GetErr(), domain.ErrInvalid) {
			t.Errorf("%d items: err = %v, want invalid", len(items), process.GetErr())
		}
	}
}

// A source written with another case reuses the name in use, in the base and in the batch; an item reports all its errors.
func TestSourceCaseAndErrors(t *testing.T) {
	db := testdb.New(t)
	ctx := testdb.Account(t, db, "test")
	first := createapplications.Process(ctx, db, []domain.NewApplication{{Company: "A", Location: "Paris", Source: "LinkedIn"}})
	first.Run()
	process := createapplications.Process(ctx, db, []domain.NewApplication{
		{Company: "B", Location: "Paris", Source: "linkedin"},
		{Company: "C", Location: "Paris", Source: "Job_Post", SourceDetail: "https://example.com"},
		{Company: "D", Location: "Paris", Source: "Indeed"},
		{Company: "E", Location: "Paris", Source: "INDEED"},
		{Company: "F", Location: "Paris", Rating: 9, AppliedAt: "2026-13-45"},
	})
	process.Run()
	result := process.GetResult()
	if first.HasErr() || process.HasErr() || result.Count != 4 || len(result.Failed) != 1 {
		t.Fatalf("result = %+v, %v %v", result, first.GetErr(), process.GetErr())
	}
	if msg := result.Failed[0].Error; !strings.Contains(msg, "rating") || !strings.Contains(msg, "date") {
		t.Errorf("error = %q, want the rating and the date", msg)
	}
	var sources []domain.Source
	if err := db.NewSelect().Model((*domain.Application)(nil)).Column("source").Order("company").Scan(ctx, &sources); err != nil {
		t.Fatal(err)
	}
	if want := []domain.Source{"LinkedIn", "LinkedIn", domain.SourceJobPost, "Indeed", "Indeed"}; !slices.Equal(sources, want) {
		t.Errorf("sources = %v, want %v", sources, want)
	}
}
