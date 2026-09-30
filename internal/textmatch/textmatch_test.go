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
	got := Normalize("«Ley 39/2015» – “procedimiento” – art.­ 21…")
	want := `"Ley 39/2015" - "procedimiento" - art. 21...`
	if got != want {
		t.Errorf("Normalize = %q, want %q", got, want)
	}
}
