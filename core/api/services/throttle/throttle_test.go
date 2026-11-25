package throttle

import (
	"context"
	"testing"
	"time"

	"github.com/cyril-fl/jat/core/services/testdb"
)

// Four tries pass, the fifth waits 30 s; a forgiven try does not count; a reset or a day forgets them.
func TestTry(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	throttle := &Throttle{db: testdb.New(t), now: func() time.Time { return now }}
	try := func() time.Duration {
		t.Helper()
		wait, err := throttle.Try(ctx, "a")
		if err != nil {
			t.Fatal(err)
		}
		return wait
	}

	for range 4 {
		if wait := try(); wait != 0 {
			t.Fatalf("wait = %s, want 0", wait)
		}
	}
	throttle.Forgive(ctx, "a")
	if wait := try(); wait != 0 {
		t.Fatalf("after forgive: wait = %s, want 0", wait)
	}
	if wait := try(); wait != 30*time.Second {
		t.Fatalf("fifth: wait = %s, want 30s", wait)
	}
	now = now.Add(10 * time.Second)
	if wait := try(); wait != 20*time.Second {
		t.Fatalf("10 s later: wait = %s, want 20s", wait)
	}
	now = now.Add(20 * time.Second)
	if wait := try(); wait != 0 {
		t.Fatalf("30 s later: wait = %s, want 0", wait)
	}
	if wait := try(); wait != time.Minute {
		t.Fatalf("sixth: wait = %s, want 1m", wait)
	}
	now = now.Add(25 * time.Hour)
	if wait := try(); wait != 0 {
		t.Fatalf("a day later: wait = %s, want 0", wait)
	}
	throttle.Reset(ctx, "a")
	if wait := try(); wait != 0 {
		t.Fatalf("after reset: wait = %s, want 0", wait)
	}
}
