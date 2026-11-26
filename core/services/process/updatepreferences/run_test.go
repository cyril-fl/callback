package updatepreferences_test

import (
	"errors"
	"testing"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/readpreferences"
	"github.com/cyril-fl/callback/core/services/process/updatepreferences"
	"github.com/cyril-fl/callback/core/services/testdb"
)

func TestRun(t *testing.T) {
	db := testdb.New(t)
	ctx := testdb.Account(t, db, "test")
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

	pro, paris, blue := domain.PlanPro, "Paris", "blue"
	update := updatepreferences.Process(ctx, db, domain.PreferenceChanges{Plan: &pro, DefaultLocation: &paris, Theme: &blue})
	update.Run()
	want := domain.Preferences{Plan: pro, Theme: blue, ColorMode: "system", DefaultLocation: paris, FollowUpDays: 7, NoReplyDays: 10, TrashDays: 30}
	if update.HasErr() || update.GetResult() != want || read() != want {
		t.Fatalf("update = %+v, read %+v, %v", update.GetResult(), read(), update.GetErr())
	}

	zero, dark := 0, "dracula"
	for _, changes := range []domain.PreferenceChanges{{FollowUpDays: &zero}, {Theme: &dark}, {ColorMode: &dark}} {
		invalid := updatepreferences.Process(ctx, db, changes)
		invalid.Run()
		if !errors.Is(invalid.GetErr(), domain.ErrInvalid) || read() != want {
			t.Errorf("%+v = %v, kept %+v", changes, invalid.GetErr(), read())
		}
	}
}
