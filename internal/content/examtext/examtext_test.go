package examtext

import (
	"strings"
	"testing"
)

// Fragments of the 2024 TAI booklet and answer key as pdftotext -layout
// prints them, shortened to a few questions per list.
const sampleBooklet = `                 No se permite la reproducción total o parcial de este cuestionario.
1.   No abra este cuestionario hasta que se le indique.
` + "\f" + `                                                          PRIMERA PARTE
1.   ¿Cuál de los siguientes derechos recogidos en el Capítulo Segundo del Título I de la Constitución Española NO forma
     parte de los Derechos Fundamentales y Libertades Públicas (Artículos 15 a 29)?
     a) Derecho a sindicarse libremente.
     b) Derecho a la propiedad privada.
     c)   Derecho a la producción y creación literaria, artística, científica y técnica.
     d) Derecho a elegir libremente su residencia.
2.   El Gobierno se rige, en su organización y funcionamiento, entre otras, por la Ley:
     a) Ley 50/1997, de 27 de noviembre.                                  b)        Ley 50/1999, de 26 de noviembre.
     c)   Ley 50/1996, de 28 de noviembre.                                d)        Ley 50/1998, de 29 de noviembre.
3.   Señale la respuesta correcta sobre las siguientes afirmaciones:
     1. La primera afirmación.
     2. La segunda afirmación.
     a) Solo la 1 es correcta.
     b) Las respuestas a) y c) son correctas.
     c)  En la misma línea, el grado más próximo al más remoto. d)        Ninguna de las anteriores.

                                                      2024 - TAI-L – MODELO A                                           Página 1 de 14
` + "\f" + `Preguntas de reserva
1.   ¿Cuál es un tipo de panel de una pantalla LCD?
     a) IPS (In-Plane Switching).                                             b)      VN (Vertical Nematic).
     c)  TA (Twisted Alignment).                                              d)      PSI (Plane Switching Input).
` + "\f" + `                                                            SUPUESTO I


El organismo en el que usted presta servicios es el órgano competente para la concesión de unas becas para personas opositoras y,
por tanto, ha surgido la necesidad de desarrollar un sistema de información.

A continuación, se expone parte del modelo de datos del sistema:

1.   ¿Qué elemento HTML utilizaría para impedir que el e-mail exceda los 100 caracteres?
     a) <input type="email" size="100">
     b) <input type="email" max="100">
     c)  <input type="email" maxlength="100">
     d) <input type="email" length="100">
2.   ¿Qué valor devuelve la siguiente función javascript?
     function prueba() {
     return 1; }
     a)   2                                                        b)     9
     c)   6                                                        d)     1
`

func TestParseBooklet(t *testing.T) {
	b, problems := ParseBooklet(sampleBooklet)
	if len(problems) > 0 {
		t.Fatalf("problems: %v", problems)
	}
	if len(b.Parts) != 2 || b.Parts[0].ID != PartGeneral || b.Parts[1].ID != PartCaseI {
		t.Fatalf("parts: %+v", b.Parts)
	}
	general := b.Parts[0].Questions
	if len(general) != 4 {
		t.Fatalf("got %d questions in the first part", len(general))
	}

	q := general[0]
	if q.Slot != (Slot{PartGeneral, false, 1}) || q.Page != 2 {
		t.Errorf("slot %v page %d", q.Slot, q.Page)
	}
	wantStem := "¿Cuál de los siguientes derechos recogidos en el Capítulo Segundo del Título I de la Constitución Española NO forma parte de los Derechos Fundamentales y Libertades Públicas (Artículos 15 a 29)?"
	if q.Stem != wantStem {
		t.Errorf("stem %q", q.Stem)
	}
	if q.Options[2] != "Derecho a la producción y creación literaria, artística, científica y técnica." {
		t.Errorf("option c %q", q.Options[2])
	}

	if got := general[1].Options; got != [4]string{"Ley 50/1997, de 27 de noviembre.", "Ley 50/1999, de 26 de noviembre.",
		"Ley 50/1996, de 28 de noviembre.", "Ley 50/1998, de 29 de noviembre."} {
		t.Errorf("two-column options %q", got)
	}

	// Numbered statements are not questions, a mention of "a) y c)" is not
	// an option, and a single space before a widely spaced letter is.
	q = general[2]
	if q.Stem != "Señale la respuesta correcta sobre las siguientes afirmaciones:\n1. La primera afirmación.\n2. La segunda afirmación." {
		t.Errorf("stem with numbered items %q", q.Stem)
	}
	if q.Options[1] != "Las respuestas a) y c) son correctas." || q.Options[3] != "Ninguna de las anteriores." {
		t.Errorf("options %q", q.Options)
	}

	if r := general[3]; r.Slot != (Slot{PartGeneral, true, 1}) || r.Options[3] != "PSI (Plane Switching Input)." {
		t.Errorf("reserve question %+v", r)
	}

	c := b.Parts[1]
	wantCase := "El organismo en el que usted presta servicios es el órgano competente para la concesión de unas becas para personas opositoras y, por tanto, ha surgido la necesidad de desarrollar un sistema de información.\n\nA continuación, se expone parte del modelo de datos del sistema:"
	if c.Case != wantCase {
		t.Errorf("case %q", c.Case)
	}
	if len(c.CasePages) != 1 || c.CasePages[0] != 4 {
		t.Errorf("case pages %v", c.CasePages)
	}
	if len(c.Questions) != 2 || c.Questions[0].Options[2] != `<input type="email" maxlength="100">` {
		t.Fatalf("case questions %+v", c.Questions)
	}
	if got := c.Questions[1].Stem; got != "¿Qué valor devuelve la siguiente función javascript?\nfunction prueba() {\nreturn 1; }" {
		t.Errorf("code keeps its lines: %q", got)
	}
}

func TestParseBookletReportsIncompleteQuestion(t *testing.T) {
	text := `PRIMERA PARTE
1.   Pregunta sin todas sus opciones:
     a) Una.
     b) Dos.
`
	_, problems := ParseBooklet(text)
	if len(problems) != 1 || problems[0].Slot.String() != "P-1" {
		t.Fatalf("problems: %v", problems)
	}
}

const sampleKey = `                            PLANTILLA DEFINITIVA DE RESPUESTAS DEL EJERCICIO ÚNICO
                                                       MODELO A


Primera parte                               3.    ANULADA                               Supuesto I
1.  b                                       4.    d                                     1.  c
2.  c                                                                                   2.  c
                                            Preguntas de reserva
                                            1.  c                                       Preguntas de reserva
                                                                                        1.  B
`

func TestParseAnswerKey(t *testing.T) {
	key, err := ParseAnswerKey(sampleKey)
	if err != nil {
		t.Fatal(err)
	}
	want := AnswerKey{
		{PartGeneral, false, 1}: 1, {PartGeneral, false, 2}: 2, {PartGeneral, false, 3}: Annulled,
		{PartGeneral, false, 4}: 3, {PartGeneral, true, 1}: 2,
		{PartCaseI, false, 1}: 2, {PartCaseI, false, 2}: 2, {PartCaseI, true, 1}: 1,
	}
	if len(key) != len(want) {
		t.Fatalf("got %v", key)
	}
	for s, v := range want {
		if key[s] != v {
			t.Errorf("%s = %d, want %d", s, key[s], v)
		}
	}
}

func TestParseAnswerKeyNeedsEveryAnswer(t *testing.T) {
	_, err := ParseAnswerKey("Primera parte\n1.  b\n2.\n")
	if err == nil || !strings.Contains(err.Error(), "P-2") {
		t.Fatalf("err = %v", err)
	}
}

func TestBuild(t *testing.T) {
	b, _ := ParseBooklet(sampleBooklet)
	final := AnswerKey{
		{PartGeneral, false, 1}: 1, {PartGeneral, false, 2}: 0, {PartGeneral, false, 3}: Annulled,
		{PartGeneral, true, 1}: 0, {PartCaseI, false, 1}: 2, {PartCaseI, false, 2}: 3,
		{PartCaseI, false, 3}: 0,
	}
	qs, problems := Build(Exam{
		Booklet: b, Final: final, Provisional: AnswerKey{{PartGeneral, false, 3}: 3},
		ImagePages: map[int]bool{4: true}, Label: "OEP 2024",
	})
	if len(qs) != 6 {
		t.Fatalf("got %d questions", len(qs))
	}
	if len(problems) != 1 || problems[0].Slot.String() != "SI-3" {
		t.Errorf("problems: %v", problems)
	}

	if q := qs[0]; q.Key != "P-1" || q.SourceRef != "OEP 2024 · nº 1" || q.Correct != "b" || q.Annulled {
		t.Errorf("first question %+v", q)
	}
	if q := qs[2]; !q.Annulled || q.Correct != "d" || !strings.Contains(q.Explanation, "anulada") {
		t.Errorf("annulled question %+v", q)
	}
	if q := qs[3]; q.Key != "P-R1" || q.SourceRef != "OEP 2024 · reserva nº 1" {
		t.Errorf("reserve question %+v", q)
	}
	q := qs[4]
	if q.Key != "SI-1" || q.SourceRef != "OEP 2024 · Supuesto I · nº 1" || q.Correct != "c" {
		t.Errorf("case question %+v", q)
	}
	if !strings.HasPrefix(q.Stem, "El organismo") || !strings.Contains(q.Stem, FiguresNote) ||
		!strings.HasSuffix(q.Stem, "exceda los 100 caracteres?") {
		t.Errorf("case question stem %q", q.Stem)
	}
}

func TestBuildAnnulledWithoutProvisional(t *testing.T) {
	b, _ := ParseBooklet("PRIMERA PARTE\n1. Pregunta:\n a) Uno\n b) Dos\n c) Tres\n d) Cuatro\n")
	qs, problems := Build(Exam{Booklet: b, Final: AnswerKey{{PartGeneral, false, 1}: Annulled}, Label: "X"})
	if len(qs) != 0 || len(problems) != 1 {
		t.Fatalf("questions %v problems %v", qs, problems)
	}
}
