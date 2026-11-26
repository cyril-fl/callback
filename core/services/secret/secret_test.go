package secret

import "testing"

func TestSealOpen(t *testing.T) {
	t.Setenv("CALLBACK_SECRET", "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=")
	sealed, err := Seal("tok")
	if err != nil || sealed == "tok" {
		t.Fatalf("seal: %q %v", sealed, err)
	}
	if plain, err := Open(sealed); err != nil || plain != "tok" {
		t.Fatalf("open: %q %v", plain, err)
	}
	if _, err := Open(sealed[:len(sealed)-4] + "AAAA"); err == nil {
		t.Error("a tampered value opens")
	}
}
