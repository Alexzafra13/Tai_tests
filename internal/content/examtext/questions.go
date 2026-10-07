// Package examtext reads the text of INAP exam PDFs, as extracted by
// pdftotext: the question booklet (with -layout) and the answer key (plain
// reading order).
package examtext

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// PartID identifies a part of the exam: the general questionnaire and each
// practical case. Question numbers restart in every part.
type PartID string

const (
	PartGeneral PartID = "P"
	PartCaseI   PartID = "SI"
	PartCaseII  PartID = "SII"
)

// Name is how the exam names the part.
func (p PartID) Name() string {
	switch p {
	case PartCaseI:
		return "Supuesto I"
	case PartCaseII:
		return "Supuesto II"
	}
	return "Primera parte"
}

// Slot is a question's position: its part, whether it is a reserve
// question (used if one of the main ones is annulled) and its number.
type Slot struct {
	Part    PartID
	Reserve bool
	Number  int
}

func (s Slot) String() string {
	r := ""
	if s.Reserve {
		r = "R"
	}
	return fmt.Sprintf("%s-%s%d", s.Part, r, s.Number)
}

type Question struct {
	Slot
	Stem    string
	Options [4]string
	Page    int
}

// Part is a section of the booklet. Case is the statement of a practical
// case, which its questions refer to.
type Part struct {
	ID        PartID
	Case      string
	CasePages []int
	Questions []Question
}

type Booklet struct {
	Parts []Part
}

// Problem is a piece of the booklet that could not be read reliably.
type Problem struct {
	Slot   Slot
	Reason string
}

func (p Problem) String() string { return p.Slot.String() + ": " + p.Reason }

var (
	pageFooter = regexp.MustCompile(`Página\s+\d+\s+de\s+\d+\s*$`)
	partHeader = regexp.MustCompile(`^(PRIMERA PARTE|SUPUESTO (I|II))$`)
	reserveHdr = regexp.MustCompile(`(?i)^(preguntas de reserva|reserva)$`)
	questionNo = regexp.MustCompile(`^\s{0,10}(\d{1,2})\.\s+\S`)
	optionMark = regexp.MustCompile(`(^\s*|\s+)([a-d])\)(\s+)`)
	spaces     = regexp.MustCompile(`\s+`)
	listItem   = regexp.MustCompile(`^[•◦●▪\-–] `)
)

// wrapWidth is the length from which a layout line is taken as wrapped
// prose and joined to the next with a space; shorter lines (code, lists),
// lines ending in a colon and list items keep their line break.
const wrapWidth = 90

// ParseBooklet reads a question booklet extracted with pdftotext -layout.
// Everything before "PRIMERA PARTE" (the instructions) is skipped.
func ParseBooklet(text string) (Booklet, []Problem) {
	p := &bookletParser{caseTxt: newCaseText()}
	for pageIdx, page := range strings.Split(text, "\f") {
		for _, line := range strings.Split(page, "\n") {
			p.line(line, pageIdx+1)
		}
	}
	p.closeQuestion()
	return Booklet{Parts: p.parts}, p.problems
}

type bookletParser struct {
	parts    []Part
	problems []Problem
	reserve  bool

	q       *Question
	field   int // -1 stem, 0-3 option
	texts   [5]textBuilder
	caseTxt textBuilder
}

func newCaseText() textBuilder { return textBuilder{paragraphs: true} }

func (p *bookletParser) part() *Part {
	if len(p.parts) == 0 {
		return nil
	}
	return &p.parts[len(p.parts)-1]
}

func (p *bookletParser) line(line string, page int) {
	trimmed := strings.TrimSpace(line)
	if pageFooter.MatchString(line) {
		return
	}
	if m := partHeader.FindStringSubmatch(trimmed); m != nil {
		p.closeQuestion()
		p.closeCase()
		id := PartGeneral
		switch m[2] {
		case "I":
			id = PartCaseI
		case "II":
			id = PartCaseII
		}
		p.parts = append(p.parts, Part{ID: id})
		p.reserve = false
		return
	}
	part := p.part()
	if part == nil {
		return
	}
	if reserveHdr.MatchString(trimmed) {
		p.closeQuestion()
		p.closeCase()
		p.reserve = true
		return
	}

	if m := questionNo.FindStringSubmatch(line); m != nil && p.startsQuestion(m[1]) {
		p.closeQuestion()
		p.closeCase()
		n, _ := strconv.Atoi(m[1])
		p.q = &Question{Slot: Slot{Part: part.ID, Reserve: p.reserve, Number: n}, Page: page}
		p.field = -1
		p.texts = [5]textBuilder{}
		line = strings.Replace(line, m[1]+".", strings.Repeat(" ", len(m[1])+1), 1)
	}

	if p.q == nil {
		// Statement of a practical case, before its first question.
		if part.ID != PartGeneral && !p.reserve {
			if trimmed != "" && (len(part.CasePages) == 0 || part.CasePages[len(part.CasePages)-1] != page) {
				part.CasePages = append(part.CasePages, page)
			}
			p.caseTxt.add(line)
		}
		return
	}

	// Options may share a line: "a) Pharming.      b) Smishing." A wide
	// gap before or after the letter tells a new option from a mention
	// such as "a) y b) son correctas".
	locs := optionMark.FindAllStringSubmatchIndex(line, -1)
	start := 0
	for _, loc := range locs {
		letter := int(line[loc[4]] - 'a')
		if letter != p.field+1 {
			continue
		}
		if loc[2] > 0 && loc[3]-loc[2] < 2 && loc[7]-loc[6] < 2 {
			continue
		}
		p.texts[p.field+1].add(line[start:loc[0]])
		p.field = letter
		start = loc[1]
	}
	p.texts[p.field+1].add(line[start:])
}

// startsQuestion reports whether a numbered line opens the next question:
// it must follow in order and the current question must be complete, so
// numbered items inside a statement are not taken for questions.
func (p *bookletParser) startsQuestion(num string) bool {
	n, _ := strconv.Atoi(num)
	if p.q == nil {
		return n == len(p.currentList())+1
	}
	return p.field == 3 && n == p.q.Number+1
}

// currentList returns the questions of the current list (main or reserve)
// of the current part.
func (p *bookletParser) currentList() []Question {
	var out []Question
	for _, q := range p.part().Questions {
		if q.Reserve == p.reserve {
			out = append(out, q)
		}
	}
	return out
}

func (p *bookletParser) closeQuestion() {
	if p.q == nil {
		return
	}
	q := *p.q
	p.q = nil
	q.Stem = p.texts[0].String()
	for i := range q.Options {
		q.Options[i] = p.texts[i+1].String()
	}
	if p.field != 3 {
		p.problems = append(p.problems, Problem{q.Slot, fmt.Sprintf("solo se han leído %d opciones", p.field+1)})
	}
	p.part().Questions = append(p.part().Questions, q)
}

func (p *bookletParser) closeCase() {
	if part := p.part(); part != nil && part.Case == "" {
		part.Case = p.caseTxt.String()
	}
	p.caseTxt = newCaseText()
}

// textBuilder joins layout lines: wrapped prose becomes one line and short
// lines (code, lists) keep their breaks. With paragraphs, blank lines
// separate paragraphs; otherwise they are page gaps and are ignored.
type textBuilder struct {
	paragraphs bool
	b          strings.Builder
	lastLen    int
	lastColon  bool
	pendingP   bool
}

func (t *textBuilder) add(raw string) {
	raw = strings.TrimRight(raw, " \t")
	text := strings.TrimSpace(spaces.ReplaceAllString(raw, " "))
	if text == "" {
		if t.paragraphs && t.b.Len() > 0 {
			t.pendingP = true
		}
		return
	}
	switch {
	case t.b.Len() == 0:
	case t.pendingP:
		t.b.WriteString("\n\n")
	case t.lastLen >= wrapWidth && !t.lastColon && !listItem.MatchString(text):
		t.b.WriteString(" ")
	default:
		t.b.WriteString("\n")
	}
	t.b.WriteString(text)
	t.lastLen = len([]rune(raw))
	t.lastColon = strings.HasSuffix(text, ":")
	t.pendingP = false
}

func (t *textBuilder) String() string { return strings.TrimSpace(t.b.String()) }
