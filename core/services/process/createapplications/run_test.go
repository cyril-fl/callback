package createapplications_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/createapplications"
	"github.com/cyril-fl/callback/core/services/testdb"
)

// The valid items of a batch are written, the invalid ones listed by index; the defaults fill what is left out.
func TestRun(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	items := []domain.NewApplication{
		{Company: " Acme ", Location: "Paris"},
		{Company: "", Location: "Lyon"},
		{Company: "Globex", Location: "Lille", Source: domain.SourceJobPost, SourceDetail: "https://jobs.example/1"},
		{Company: "Initech", Location: "Nantes", Source: domain.SourceRecruiter, SourceDetail: "not an e-mail"},
		{Company: "Umbrella", Location: "Brest", Rating: 6},
		{Company: "Hooli", Location: "Rennes", Source: domain.SourceRecruiter, SourceDetail: "jane@hooli.example", Comment: "<b>raw</b>"},
		{Company: "Vandelay", Location: "Lyon", Street: " 1 rue de la Paix ", PostalCode: "69001", Country: "France", Source: " Agence intérim Adecco ", SourceDetail: "Marie, 06 00 00 00 00"},
		{Company: "Pied Piper", Location: "Paris", Source: domain.Source(strings.Repeat("x", domain.MaxSource+1))},
		{Company: "Dunder", Location: "Scranton", Source: domain.SourceSpontaneous, SourceDetail: "not for spontaneous"},
		{Company: "Wonka", Location: "Paris", Website: "jobs@wonka.example"},
		{Company: "Krusty", Location: "Paris", Website: "not a site"},
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
	wants := map[int]error{1: domain.ErrNoCompany, 3: domain.ErrInvalidDetail, 4: domain.ErrInvalidRating, 7: domain.ErrInvalidSource, 8: domain.ErrInvalidDetail, 10: domain.ErrInvalidWebsite}
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
	if err := db.NewSelect().Model(&vandelay).Where("id = ?", result.IDs[3]).Scan(ctx); err != nil || vandelay.Source != "Agence intérim Adecco" || vandelay.Street != "1 rue de la Paix" || vandelay.Country != "France" {
		t.Errorf("vandelay = %+v, %v", vandelay, err)
	}
	var hooli domain.Application
	if err := db.NewSelect().Model(&hooli).Where("id = ?", result.IDs[2]).Scan(ctx); err != nil || hooli.Comment != "<b>raw</b>" {
		t.Errorf("comment = %q, %v; want it stored raw", hooli.Comment, err)
	}
}

func TestLimits(t *testing.T) {
	db := testdb.New(t)
	for _, items := range [][]domain.NewApplication{nil, make([]domain.NewApplication, domain.MaxBatch+1)} {
		process := createapplications.Process(context.Background(), db, items)
		process.Run()
		if !errors.Is(process.GetErr(), domain.ErrInvalid) {
			t.Errorf("%d items: err = %v, want invalid", len(items), process.GetErr())
		}
	}
}
