package liststatuschanges_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

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

	history := func(input liststatuschanges.Input) []string {
		t.Helper()
		process := liststatuschanges.Process(ctx, db, input)
		process.Run()
		if process.HasErr() {
			t.Fatal(process.GetErr())
		}
		var got []string
		for _, change := range process.GetResult() {
			got = append(got, change.ApplicationID+" "+change.Company+" "+string(change.From)+">"+string(change.Status))
		}
		return got
	}
	want := []string{ids[0] + " Acme >applied", ids[1] + " Globex >interview", ids[0] + " Acme applied>responded"}
	if got := history(liststatuschanges.Input{}); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("history = %v, want %v", got, want)
	}
	if got := history(liststatuschanges.Input{IDs: []string{ids[0]}}); fmt.Sprint(got) != fmt.Sprint([]string{want[0], want[2]}) {
		t.Errorf("history of %s = %v", ids[0], got)
	}
	tomorrow := time.Now().UTC().AddDate(0, 0, 1).Format(domain.DateLayout)
	if got := history(liststatuschanges.Input{Since: tomorrow}); len(got) != 0 {
		t.Errorf("history since %s = %v, want none", tomorrow, got)
	}
}

func TestInvalidSince(t *testing.T) {
	process := liststatuschanges.Process(context.Background(), testdb.New(t), liststatuschanges.Input{Since: "yesterday"})
	process.Run()
	if !errors.Is(process.GetErr(), liststatuschanges.ErrInvalidSince) {
		t.Errorf("err = %v, want ErrInvalidSince", process.GetErr())
	}
}
