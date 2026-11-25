package readstats_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/readstats"
	"github.com/cyril-fl/callback/core/services/seed"
	"github.com/cyril-fl/callback/core/services/testdb"
)

func TestRun(t *testing.T) {
	db := testdb.New(t)
	empty := readstats.Process(context.Background(), db, domain.Filter{})
	empty.Run()
	if got := empty.GetResult(); empty.HasErr() || got.Total != 0 || got.ResponseRate != nil || len(got.ByStatus) != len(domain.Statuses) {
		t.Fatalf("empty = %+v, %v", got, empty.GetErr())
	}

	seed.Applications(t, db,
		domain.NewApplication{Company: "A", Location: "Paris"},
		domain.NewApplication{Company: "B", Location: "Paris", Source: "LinkedIn"},
		domain.NewApplication{Company: "C", Location: "Paris", Status: domain.StatusInterview},
		domain.NewApplication{Company: "D", Location: "Paris", Status: domain.StatusRejected},
	)
	process := readstats.Process(context.Background(), db, domain.Filter{})
	process.Run()
	got := process.GetResult()
	if process.HasErr() || got.Total != 4 || got.ByStatus[domain.StatusApplied] != 2 || got.ByStatus[domain.StatusOffer] != 0 || got.ResponseRate == nil || *got.ResponseRate != 0.5 ||
		got.BySource["LinkedIn"] != 1 || got.BySource[domain.SourceSpontaneous] != 3 ||
		got.ResponseRateBySource["LinkedIn"] != 0 || got.ResponseRateBySource[domain.SourceSpontaneous] != 2.0/3 {
		t.Errorf("stats = %+v, %v", got, process.GetErr())
	}

	recent := readstats.Process(context.Background(), db, domain.Filter{AppliedAfter: "2026-01-31", Source: "linkedin"})
	recent.Run()
	if got := recent.GetResult(); recent.HasErr() || got.Total != 1 || got.BySource["LinkedIn"] != 1 {
		t.Errorf("recent = %+v, %v", got, recent.GetErr())
	}
	invalid := readstats.Process(context.Background(), db, domain.Filter{AppliedBefore: "soon"})
	if invalid.Run(); !errors.Is(invalid.GetErr(), domain.ErrInvalid) {
		t.Errorf("invalid filter: err = %v", invalid.GetErr())
	}
}
