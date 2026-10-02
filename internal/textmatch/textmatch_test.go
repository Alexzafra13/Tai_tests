package textmatch

import "testing"

const article = "Artículo 21. Obligación de resolver.\n\n1. La Administración está obligada a dictar\nresolución expresa y a notificarla en todos los procedimientos cualquiera que sea su forma de iniciación.\n\n2. El plazo máximo en el que debe notificarse la resolución expresa será el fijado por la norma reguladora del correspondiente procedimiento. Este plazo no podrá exceder de seis meses salvo que una norma con rango de Ley establezca uno mayor o así venga previsto en el Derecho de la Unión Europea."

func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		quote string
		want  bool
	}{
		{"exact", "Este plazo no podrá exceder de seis meses", true},
		{"across line breaks and nbsp", "a dictar resolución expresa y a notificarla en todos los procedimientos", true},
		{"extra spaces in quote", "  Este   plazo\tno podrá  exceder de seis meses ", true},
		{"typographic quotes normalized on both sides", "Obligación de resolver.", true},
		{"changed word", "Este plazo no podrá exceder de tres meses", false},
		{"case differs", "este plazo no podrá exceder de seis meses", false},
		{"paraphrase", "El plazo máximo es de seis meses", false},
		{"elided fragment", "La Administración [...] resolución expresa", false},
		{"empty", "   ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Contains(article, tt.quote); got != tt.want {
				t.Errorf("Contains(%q) = %v, want %v", tt.quote, got, tt.want)
			}
		})
	}
}

func TestNormalizePunctuation(t *testing.T) {
	got := Normalize("«Ley 39/2015» – “procedimiento” – art.\u00ad 21…")
	want := `"Ley 39/2015" - "procedimiento" - art. 21...`
	if got != want {
		t.Errorf("Normalize = %q, want %q", got, want)
	}
}

func TestLocate(t *testing.T) {
	ex, ok := Locate(article, "Este plazo no podrá exceder de seis meses", 40)
	if !ok {
		t.Fatal("quote not located")
	}
	if ex.Match != "Este plazo no podrá exceder de seis meses" {
		t.Errorf("match = %q", ex.Match)
	}
	if ex.Before != "del correspondiente procedimiento. " || !ex.ClippedStart {
		t.Errorf("before = %q clipped=%v", ex.Before, ex.ClippedStart)
	}
	if ex.After != " salvo que una norma con rango de Ley" || !ex.ClippedEnd {
		t.Errorf("after = %q clipped=%v", ex.After, ex.ClippedEnd)
	}

	ex, ok = Locate(article, "Artículo 21. Obligación de resolver.", 1000)
	if !ok || ex.Before != "" || ex.ClippedStart || ex.ClippedEnd {
		t.Errorf("whole-text window: %+v", ex)
	}
	if _, ok := Locate(article, "no existe en el texto", 10); ok {
		t.Error("missing quote located")
	}
}
