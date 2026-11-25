package apiclient

import (
	"os"
	"testing"

	"github.com/cyril-fl/jat/config"
)

// TestSession keeps the file readable by the user only, and lets JAT_URL and JAT_SESSION override it one by one.
func TestSession(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(config.Env("URL"), "")
	t.Setenv(config.Env("SESSION"), "")
	if err := (Session{URL: "http://saved", Token: "saved"}).Save(); err != nil {
		t.Fatal(err)
	}
	file, _ := SessionFile()
	if info, err := os.Stat(file); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("session file: %v, %v; want 0600", info, err)
	}
	if got := Read(); got != (Session{URL: "http://saved", Token: "saved"}) {
		t.Errorf("Read = %+v", got)
	}
	t.Setenv(config.Env("SESSION"), "env")
	if got := Read(); got != (Session{URL: "http://saved", Token: "env"}) {
		t.Errorf("Read with JAT_SESSION = %+v", got)
	}
	if _, err := (Session{URL: "http://x"}).Client(); err != ErrNoSession {
		t.Errorf("Client without token: %v, want ErrNoSession", err)
	}
}
