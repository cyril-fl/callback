package createuser_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cyril-fl/callback/core/services/process/createuser"
	"github.com/cyril-fl/callback/core/services/testdb"
)

// The first account of a base needs no invitation and becomes admin; the next ones need one, but in the public mode,
// and are not admins.
func TestFirstAccountIsAdmin(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	first := createuser.Process(ctx, db, createuser.Input{Name: "cyril", Password: "long enough"}, false)
	first.Run()
	if first.HasErr() || !first.GetResult().Admin {
		t.Fatalf("first: %+v %v", first.GetResult(), first.GetErr())
	}
	invited := createuser.Process(ctx, db, createuser.Input{Name: "bob", Password: "long enough"}, false)
	invited.Run()
	if !errors.Is(invited.GetErr(), createuser.ErrBadInvite) {
		t.Errorf("second without invitation: %v, want %v", invited.GetErr(), createuser.ErrBadInvite)
	}
	public := createuser.Process(ctx, db, createuser.Input{Name: "bob", Password: "long enough"}, true)
	public.Run()
	if public.HasErr() || public.GetResult().Admin {
		t.Errorf("public: %+v %v", public.GetResult(), public.GetErr())
	}
	for _, input := range []createuser.Input{{Name: "  ", Password: "long enough"}, {Name: string(make([]rune, 65)), Password: "long enough"}, {Name: "Bob", Password: "long enough"}} {
		process := createuser.Process(ctx, db, input, true)
		process.Run()
		if !process.HasErr() {
			t.Errorf("%q: want an error", input.Name)
		}
	}
}
