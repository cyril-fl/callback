package readstats_test

import (
	"context"
	"testing"

	"github.com/cyril-fl/jat/core/domain"
	"github.com/cyril-fl/jat/core/services/process/readstats"
	"github.com/cyril-fl/jat/core/services/seed"
	"github.com/cyril-fl/jat/core/services/testdb"
)

func TestRun(t *testing.T) {
	db := testdb.New(t)
	empty := readstats.Process(context.Background(), db)
	empty.Run()
	if got := empty.GetResult(); empty.HasErr() || got.Total != 0 || got.ResponseRate != 0 || len(got.ByStatus) != len(domain.Statuses) {
		t.Fatalf("empty = %+v, %v", got, empty.GetErr())
	}

	seed.Applications(t, db,
		domain.NewApplication{Company: "A", Location: "Paris"},
		domain.NewApplication{Company: "B", Location: "Paris"},
		domain.NewApplication{Company: "C", Location: "Paris", Status: domain.StatusInterview},
		domain.NewApplication{Company: "D", Location: "Paris", Status: domain.StatusRejected},
	)
	process := readstats.Process(context.Background(), db)
	process.Run()
	got := process.GetResult()
	if process.HasErr() || got.Total != 4 || got.ByStatus[domain.StatusApplied] != 2 || got.ByStatus[domain.StatusOffer] != 0 || got.ResponseRate != 0.5 {
		t.Errorf("stats = %+v, %v", got, process.GetErr())
	}
}
