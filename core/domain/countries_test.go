package domain

import (
	"errors"
	"testing"
)

// A country is stored as its code, given by its code or English name whatever the case; anything else is refused.
func TestCountry(t *testing.T) {
	for given, want := range map[string]string{"IT": "IT", " it ": "IT", "italy": "IT", "Côte d’Ivoire": "CI", "": ""} {
		a := Application{Company: "Acme", Location: "Roma", Country: given}
		a.Normalize()
		if a.Country != want || a.Validate() != nil {
			t.Errorf("%q: country %q, %v", given, a.Country, a.Validate())
		}
	}
	a := Application{Company: "Acme", Location: "Roma", Country: "Italie"}
	a.Normalize()
	if err := a.Validate(); !errors.Is(err, ErrInvalidCountry) {
		t.Errorf("Italie: %v", err)
	}
	if len(Countries) != 249 {
		t.Errorf("%d countries", len(Countries))
	}
}

// An unknown country of an older base or backup goes to the comment, not lost.
func TestKeepCountry(t *testing.T) {
	a := Application{Country: "Italie", Comment: "Call back"}
	if !a.KeepCountry() || a.Country != "" || a.Comment != "Country: Italie\nCall back" {
		t.Errorf("unknown = %+v", a)
	}
	a = Application{Country: "France"}
	if !a.KeepCountry() || a.Country != "FR" {
		t.Errorf("known = %+v", a)
	}
	if a.KeepCountry() {
		t.Error("a code changed")
	}
}
