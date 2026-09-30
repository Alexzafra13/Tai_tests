package quiz

import (
	"math/rand/v2"
	"testing"
)

// reverse is a deterministic "shuffle" for tests.
func reverse(n int, swap func(i, j int)) {
	for i := 0; i < n/2; i++ {
		swap(i, n-1-i)
	}
}

func TestNewOptionOrder(t *testing.T) {
	tests := []struct {
		name    string
		options [4]string
		fixed   bool
		want    optionOrder
	}{
		{"plain options move", [4]string{"Tres meses", "Seis meses", "Un año", "Dos meses"}, false, optionOrder{3, 2, 1, 0}},
		{"fixed flag", [4]string{"Tres meses", "Seis meses", "Un año", "Dos meses"}, true, identityOrder},
		{"all of the above stays last", [4]string{"HTTP", "FTP", "SMTP", "Todas las anteriores son correctas"}, false, optionOrder{2, 1, 0, 3}},
		{"none of the above stays", [4]string{"Ninguna es correcta", "TCP", "UDP", "ICMP"}, false, optionOrder{0, 3, 2, 1}},
		{"letter reference freezes all", [4]string{"El Rey", "Las Cortes", "A y B son correctas", "El Gobierno"}, false, identityOrder},
		{"letter reference with parens", [4]string{"x", "y", "z", "Las respuestas a) y c) son correctas"}, false, identityOrder},
		{"single letter reference", [4]string{"x", "y", "z", "Solo la A es correcta"}, false, identityOrder},
		{"article before a word is not a letter", [4]string{"La administración", "La autonomía", "Las bases", "Las cortes"}, false, optionOrder{3, 2, 1, 0}},
		{"letter list", [4]string{"x", "y", "z", "Son correctas la b, c"}, false, identityOrder},
		{"two relative options", [4]string{"Ambas son correctas", "Ninguna de las anteriores", "Uno", "Dos"}, false, optionOrder{0, 1, 3, 2}},
		{"words with a y b letters inside", [4]string{"Tabla y base", "Clave y valor", "Árbol", "Grafo"}, false, optionOrder{3, 2, 1, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newOptionOrder(tt.options, tt.fixed, reverse); got != tt.want {
				t.Errorf("order = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOptionOrderMapping(t *testing.T) {
	o := optionOrder{2, 0, 3, 1}
	opts := [4]string{"a", "b", "c", "d"}
	if got := o.display(opts); got != [4]string{"c", "a", "d", "b"} {
		t.Errorf("display = %v", got)
	}
	for orig := range 4 {
		if o.toOriginal(o.toDisplay(orig)) != orig {
			t.Errorf("round trip failed for %d", orig)
		}
	}
	if parseOptionOrder(o.String()) != o {
		t.Error("String/parse round trip failed")
	}
	for _, bad := range []string{"", "0012", "01234", "0x23"} {
		if parseOptionOrder(bad) != identityOrder {
			t.Errorf("parse(%q) should fall back to identity", bad)
		}
	}
}

// With the real shuffle every order must be a permutation that keeps
// pinned options in place.
func TestRandomOrderIsValid(t *testing.T) {
	opts := [4]string{"HTTP", "FTP", "SMTP", "Todas las anteriores"}
	seen := map[optionOrder]bool{}
	for range 200 {
		o := newOptionOrder(opts, false, rand.Shuffle)
		if o[3] != 3 {
			t.Fatalf("pinned option moved: %v", o)
		}
		if parseOptionOrder(o.String()) != o {
			t.Fatalf("not a permutation: %v", o)
		}
		seen[o] = true
	}
	if len(seen) < 4 {
		t.Errorf("only %d distinct orders in 200 shuffles; expected up to 6", len(seen))
	}
}
