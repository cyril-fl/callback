package updatepreferences_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/readpreferences"
	"github.com/cyril-fl/callback/core/services/process/updatepreferences"
	"github.com/cyril-fl/callback/core/services/testdb"
)

func TestRun(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	read := func() domain.Preferences {
		process := readpreferences.Process(ctx, db)
		process.Run()
		if process.HasErr() {
			t.Fatal(process.GetErr())
		}
		return process.GetResult()
	}
	if got := read(); got != domain.DefaultPreferences {
		t.Fatalf("empty base = %+v", got)
	}

	pro, paris := domain.PlanPro, "Paris"
	update := updatepreferences.Process(ctx, db, domain.PreferenceChanges{Plan: &pro, DefaultLocation: &paris})
	update.Run()
	want := domain.Preferences{Plan: pro, DefaultLocation: paris, FollowUpDays: 7, NoReplyDays: 10, TrashDays: 30}
	if update.HasErr() || update.GetResult() != want || read() != want {
		t.Fatalf("update = %+v, read %+v, %v", update.GetResult(), read(), update.GetErr())
	}

	zero := 0
	invalid := updatepreferences.Process(ctx, db, domain.PreferenceChanges{FollowUpDays: &zero})
	invalid.Run()
	if !errors.Is(invalid.GetErr(), domain.ErrInvalid) || read() != want {
		t.Errorf("follow_up_days 0 = %v, kept %+v", invalid.GetErr(), read())
	}
}
