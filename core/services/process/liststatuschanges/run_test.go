package liststatuschanges_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/deleteapplications"
	"github.com/cyril-fl/callback/core/services/process/liststatuschanges"
	"github.com/cyril-fl/callback/core/services/process/updateapplications"
	"github.com/cyril-fl/callback/core/services/seed"
	"github.com/cyril-fl/callback/core/services/testdb"
)

func ptr[T any](value T) *T { return &value }

// The history is written by the create and by a change of status only, and goes with the application.
func TestHistory(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	ids := seed.Applications(t, db,
		domain.NewApplication{Company: "Acme", Location: "Paris"},
		domain.NewApplication{Company: "Globex", Location: "Lyon", Status: domain.StatusInterview},
		domain.NewApplication{Company: "Initech", Location: "Lille"},
	)
	update := updateapplications.Process(ctx, db, updateapplications.Input{Ops: []updateapplications.Op{
		{Filter: domain.Filter{IDs: []string{ids[0]}}, Set: domain.Changes{Status: ptr(domain.StatusResponded)}},
		{Filter: domain.Filter{IDs: []string{ids[1]}}, Set: domain.Changes{Status: ptr(domain.StatusInterview), Rating: ptr(3)}},
	}})
	update.Run()
	remove := deleteapplications.Process(ctx, db, deleteapplications.Input{Filter: domain.Filter{IDs: []string{ids[2]}}})
	remove.Run()
	if update.HasErr() || remove.HasErr() {
		t.Fatal(update.GetErr(), remove.GetErr())
	}

	process := liststatuschanges.Process(ctx, db)
	process.Run()
	if process.HasErr() {
		t.Fatal(process.GetErr())
	}
	var got []string
	for _, change := range process.GetResult() {
		got = append(got, change.ApplicationID+" "+string(change.Status))
	}
	want := []string{ids[0] + " applied", ids[1] + " interview", ids[0] + " responded"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("history = %v, want %v", got, want)
	}
}
