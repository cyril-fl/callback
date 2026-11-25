package purgeapplications_test

import (
	"context"
	"testing"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/purgeapplications"
	"github.com/cyril-fl/callback/core/services/seed"
	"github.com/cyril-fl/callback/core/services/testdb"
)

// The applications and their history go; the rest of the base stays.
func TestRun(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	seed.Applications(t, db,
		domain.NewApplication{Company: "Acme", Location: "Paris"},
		domain.NewApplication{Company: "Globex", Location: "Lyon"},
	)
	process := purgeapplications.Process(ctx, db)
	process.Run()
	if process.HasErr() || process.GetResult() != 2 {
		t.Fatalf("purge = %d, %v; want 2", process.GetResult(), process.GetErr())
	}
	for _, model := range []any{(*domain.Application)(nil), (*domain.StatusChange)(nil)} {
		if count, err := db.NewSelect().Model(model).Count(ctx); err != nil || count != 0 {
			t.Errorf("%T left: %d, %v", model, count, err)
		}
	}
}
