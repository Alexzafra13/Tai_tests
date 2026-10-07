package examtext

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/alexzafra13/tai_tests/internal/content"
)

// FiguresNote is added to a practical case whose pages have images: the
// text extraction keeps the words but not diagrams or screenshots.
const FiguresNote = "(El enunciado original incluye figuras que no se reproducen aquí.)"

// DiagramNote is added to a practical case with a diagram drawn as text,
// whose words come out of order.
const DiagramNote = "(El diagrama del enunciado original no se reproduce fielmente aquí: consúltalo en el cuestionario del INAP.)"

// textDiagram spots the leftovers of a class or entity diagram drawn with
// text: attribute markers ("+IdSolicitud") and multiplicities ("1..N").
var textDiagram = regexp.MustCompile(`(?m)(^|\s)\+[A-Z]\w+.*\b[01]\.\.[1N*]\b|\b[01]\.\.[1N*]\b.*(^|\s)\+[A-Z]\w+`)

// Exam is what Build needs to turn a booklet into bank questions.
type Exam struct {
	Booklet Booklet
	// Final is the definitive answer key, the one that counts.
	Final AnswerKey
	// Provisional gives the answer of the questions the final key annuls,
	// which still need one to be stored; optional.
	Provisional AnswerKey
	// ImagePages are the booklet pages (1-based) that contain images.
	ImagePages map[int]bool
	// Label starts every source_ref, e.g. "OEP 2024".
	Label string
}

// Build matches every question with its answer. Questions that cannot be
// matched are left out and reported.
func Build(e Exam) ([]content.BankQuestion, []Problem) {
	var out []content.BankQuestion
	var problems []Problem
	seen := map[Slot]bool{}
	for _, part := range e.Booklet.Parts {
		context := part.Case
		switch {
		case context == "":
		case slices.ContainsFunc(part.CasePages, func(p int) bool { return e.ImagePages[p] }):
			context += "\n\n" + FiguresNote
		case textDiagram.MatchString(context):
			context += "\n\n" + DiagramNote
		}
		for _, q := range part.Questions {
			seen[q.Slot] = true
			answer, ok := e.Final[q.Slot]
			if !ok {
				problems = append(problems, Problem{q.Slot, "no aparece en la plantilla definitiva"})
				continue
			}
			bq := content.BankQuestion{
				Key:       q.Slot.String(),
				SourceRef: e.Label + " · " + describe(q.Slot),
				Stem:      q.Stem,
				Options:   q.Options,
			}
			if context != "" {
				bq.Stem = context + "\n\n" + q.Stem
			}
			if answer == Annulled {
				prov, ok := e.Provisional[q.Slot]
				if !ok || prov == Annulled {
					problems = append(problems, Problem{q.Slot, "anulada y sin respuesta en la plantilla provisional"})
					continue
				}
				answer = prov
				bq.Annulled = true
				bq.Explanation = fmt.Sprintf("Pregunta anulada en la plantilla definitiva. La plantilla provisional daba como correcta la %c).", 'a'+prov)
			}
			bq.Correct = string(rune('a' + answer))
			if slices.Contains(bq.Options[:], "") || bq.Stem == "" {
				problems = append(problems, Problem{q.Slot, "enunciado u opción vacíos"})
				continue
			}
			out = append(out, bq)
		}
	}
	for s := range e.Final {
		if !seen[s] {
			problems = append(problems, Problem{s, "está en la plantilla pero no en el cuestionario"})
		}
	}
	slices.SortFunc(problems, func(a, b Problem) int { return compareSlots(a.Slot, b.Slot) })
	return out, problems
}

// describe is the human reference of a slot: "nº 37", "Supuesto I ·
// reserva nº 2".
func describe(s Slot) string {
	ref := fmt.Sprintf("nº %d", s.Number)
	if s.Reserve {
		ref = "reserva " + ref
	}
	if s.Part != PartGeneral {
		ref = s.Part.Name() + " · " + ref
	}
	return ref
}

func compareSlots(a, b Slot) int {
	order := map[PartID]int{PartGeneral: 0, PartCaseI: 1, PartCaseII: 2}
	if d := order[a.Part] - order[b.Part]; d != 0 {
		return d
	}
	if a.Reserve != b.Reserve {
		if a.Reserve {
			return 1
		}
		return -1
	}
	return a.Number - b.Number
}
