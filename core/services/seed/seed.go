// Package seed creates applications for the tests.
package seed

import (
	"context"
	"testing"

	"github.com/cyril-fl/jat/core/domain"
	"github.com/cyril-fl/jat/core/services/process/createapplications"
	"github.com/uptrace/bun"
)

// Applications creates items and gives their ids, in order; a failed item fails t.
func Applications(t testing.TB, db bun.IDB, items ...domain.NewApplication) []string {
	t.Helper()
	process := createapplications.Process(context.Background(), db, items)
	process.Run()
	if process.HasErr() || len(process.GetResult().Failed) > 0 {
		t.Fatalf("seed: %v %+v", process.GetErr(), process.GetResult().Failed)
	}
	return process.GetResult().IDs
}
