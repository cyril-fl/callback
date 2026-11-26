// Package seed creates applications for the tests.
package seed

import (
	"context"
	"testing"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/createapplications"
	"github.com/uptrace/bun"
)

// Applications creates items for the account of ctx and gives their ids, in order; a failed item fails t.
func Applications(t testing.TB, ctx context.Context, db bun.IDB, items ...domain.NewApplication) []string {
	t.Helper()
	process := createapplications.Process(ctx, db, items)
	process.Run()
	if process.HasErr() || len(process.GetResult().Failed) > 0 {
		t.Fatalf("seed: %v %+v", process.GetErr(), process.GetResult().Failed)
	}
	return process.GetResult().IDs
}
