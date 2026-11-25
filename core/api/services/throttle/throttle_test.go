package throttle

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/cyril-fl/callback/core/services/testdb"
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

// An IPv6 client is counted by its /64: it holds the whole of it.
func TestAddressKey(t *testing.T) {
	for remote, want := range map[string]string{
		"192.0.2.1:1234":              "address 192.0.2.1",
		"[2001:db8:1:2:3:4:5:6]:1234": "address 2001:db8:1:2::/64",
		"[2001:db8:1:2:ffff::]:80":    "address 2001:db8:1:2::/64",
	} {
		if got := AddressKey(&http.Request{RemoteAddr: remote}); got != want {
			t.Errorf("%s: %q, want %q", remote, got, want)
		}
	}
}
