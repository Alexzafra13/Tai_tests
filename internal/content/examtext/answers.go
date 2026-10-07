package examtext

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Annulled marks a question the answer key cancels.
const Annulled = -1

// AnswerKey maps each question to its correct option (0-3) or Annulled.
type AnswerKey map[Slot]int

var (
	keyPart   = regexp.MustCompile(`(?i)^(primera parte|supuesto (i|ii))$`)
	keyNumber = regexp.MustCompile(`^(\d{1,2})\.$`)
	// The final key may flag an answer changed after appeals: "d MODIFICADA".
	keyAnswer  = regexp.MustCompile(`^([a-dA-D]|ANULADA)(?:\s+\(?MODIFICADA\)?)?$`)
	keyNumAnsw = regexp.MustCompile(`^(\d{1,2})\.\s+([a-dA-D]|ANULADA)(?:\s+\(?MODIFICADA\)?)?$`)
)

// ParseAnswerKey reads an answer key extracted with pdftotext -layout.
// The key is printed in columns, read one after the other; within a
// column, answers are matched in order to the pending question numbers,
// so "71." and "b" may come in separate pieces.
func ParseAnswerKey(text string) (AnswerKey, error) {
	key := AnswerKey{}
	var part PartID
	reserve := false
	var pending []Slot

	flush := func() error {
		if len(pending) > 0 {
			return fmt.Errorf("plantilla: %d preguntas sin respuesta, desde %s", len(pending), pending[0])
		}
		return nil
	}
	answer := func(tok string) error {
		if len(pending) == 0 {
			return fmt.Errorf("plantilla: respuesta %q sin número de pregunta", tok)
		}
		v := Annulled
		if tok != "ANULADA" {
			v = int(strings.ToLower(tok)[0] - 'a')
		}
		key[pending[0]] = v
		pending = pending[1:]
		return nil
	}
	number := func(tok string) error {
		if part == "" {
			return fmt.Errorf("plantilla: pregunta %s fuera de una parte", tok)
		}
		n, _ := strconv.Atoi(tok)
		s := Slot{Part: part, Reserve: reserve, Number: n}
		if _, dup := key[s]; dup {
			return fmt.Errorf("plantilla: %s repetida", s)
		}
		pending = append(pending, s)
		return nil
	}

	for _, line := range columnPieces(text) {
		if m := keyPart.FindStringSubmatch(line); m != nil {
			if err := flush(); err != nil {
				return nil, err
			}
			part, reserve = PartGeneral, false
			switch strings.ToUpper(m[2]) {
			case "I":
				part = PartCaseI
			case "II":
				part = PartCaseII
			}
			continue
		}
		if reserveHdr.MatchString(line) {
			if err := flush(); err != nil {
				return nil, err
			}
			reserve = true
			continue
		}
		var err error
		switch {
		case keyNumAnsw.MatchString(line):
			m := keyNumAnsw.FindStringSubmatch(line)
			if err = number(m[1]); err == nil {
				err = answer(m[2])
			}
		case keyNumber.MatchString(line):
			err = number(keyNumber.FindStringSubmatch(line)[1])
		case keyAnswer.MatchString(line):
			err = answer(keyAnswer.FindStringSubmatch(line)[1])
		}
		if err != nil {
			return nil, err
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	if len(key) == 0 {
		return nil, fmt.Errorf("plantilla: no se ha encontrado ninguna respuesta")
	}
	return key, nil
}

var columnGap = regexp.MustCompile(`\S+( \S+)*`)

// columnPieces splits a layout page into the pieces of text separated by
// two or more spaces and returns them column by column. Pieces whose start
// positions are close belong to the same column.
func columnPieces(text string) []string {
	var out []string
	for _, page := range strings.Split(text, "\f") {
		type piece struct {
			line, col int
			text      string
		}
		var pieces []piece
		var starts []int
		for i, line := range strings.Split(page, "\n") {
			runes := []rune(line)
			for _, loc := range columnGap.FindAllStringIndex(string(runes), -1) {
				col := len([]rune(string(runes)[:loc[0]]))
				pieces = append(pieces, piece{i, col, string(runes)[loc[0]:loc[1]]})
				starts = append(starts, col)
			}
		}
		slices.Sort(starts)
		starts = slices.Compact(starts)
		column := map[int]int{}
		c := 0
		for i, st := range starts {
			if i > 0 && st-starts[i-1] > 12 {
				c++
			}
			column[st] = c
		}
		slices.SortStableFunc(pieces, func(a, b piece) int {
			if d := column[a.col] - column[b.col]; d != 0 {
				return d
			}
			return a.line - b.line
		})
		for _, p := range pieces {
			out = append(out, p.text)
		}
	}
	return out
}
