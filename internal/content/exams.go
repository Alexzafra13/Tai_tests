package content

import (
	"context"
	"regexp"
	"sort"
	"strconv"
)

// An official exam as a student takes it: a first part and the practical
// cases, each followed by its reserve questions. As in the exam's rules,
// reserve questions replace, in their order, the questions that cannot be
// answered (annulled, or not published in this installation), so each part
// keeps the number of questions it had.
type Exam struct {
	ID        int64      `json:"id"`
	Title     string     `json:"title"`
	Reference string     `json:"reference"`
	URL       string     `json:"url"`
	Parts     []ExamPart `json:"parts"`
}

type ExamPart struct {
	// Name is "Primera parte" or the case as written in the exam
	// ("Supuesto I").
	Name string `json:"name"`
	// Case marks a practical case; the candidate answers only one of them.
	Case bool `json:"case"`
	// QuestionIDs are the questions to answer, in exam order.
	QuestionIDs []int64 `json:"question_ids"`
	// Replaced counts the questions left out and taken over by reserves.
	Replaced int `json:"replaced"`
	// Annulled and Unpublished say why questions were left out: annulled by
	// the INAP's answer key, or not published in this installation (such as
	// those that need a figure the text cannot show).
	Annulled    int `json:"annulled"`
	Unpublished int `json:"unpublished"`
	// Missing counts those left out with no reserve to take over.
	Missing int `json:"missing"`
}

const firstPart = "Primera parte"

// examRef reads the references the exam importer writes: "OEP 2024 · nº 3",
// "OEP 2024 · reserva nº 1", "OEP 2024 · Supuesto II · nº 7".
var examRef = regexp.MustCompile(`(?:· (Supuesto [IVX]+) )?· (reserva )?nº (\d+)$`)

type examQuestion struct {
	id       int64
	part     string
	reserve  bool
	number   int
	eligible bool
	annulled bool
}

// Exams returns the official exams that have at least one question a test
// can include.
func (s *Store) Exams(ctx context.Context) ([]Exam, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT s.id, s.title, s.reference, s.url, q.id, q.source_ref,
			q.status = 'published' AND q.annulled = 0, q.annulled
		FROM sources s JOIN questions q ON q.source_id = s.id
		WHERE s.kind = ? ORDER BY s.reference DESC, q.id`, KindINAPExam)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var exams []Exam
	byExam := map[int64][]examQuestion{}
	for rows.Next() {
		var e Exam
		var q examQuestion
		var ref string
		if err := rows.Scan(&e.ID, &e.Title, &e.Reference, &e.URL, &q.id, &ref, &q.eligible, &q.annulled); err != nil {
			return nil, err
		}
		m := examRef.FindStringSubmatch(ref)
		if m == nil {
			// Not from the exam paper (added by hand): not part of it.
			continue
		}
		q.part, q.reserve = m[1], m[2] != ""
		q.number, _ = strconv.Atoi(m[3])
		if q.part == "" {
			q.part = firstPart
		}
		if _, ok := byExam[e.ID]; !ok {
			exams = append(exams, e)
		}
		byExam[e.ID] = append(byExam[e.ID], q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := []Exam{}
	for _, e := range exams {
		e.Parts = examParts(byExam[e.ID])
		if len(e.Parts) > 0 {
			out = append(out, e)
		}
	}
	return out, nil
}

func examParts(qs []examQuestion) []ExamPart {
	sort.SliceStable(qs, func(i, j int) bool {
		a, b := qs[i], qs[j]
		if a.part != b.part {
			// "Primera parte" sorts before "Supuesto …".
			return a.part < b.part
		}
		if a.reserve != b.reserve {
			return !a.reserve
		}
		return a.number < b.number
	})
	var parts []ExamPart
	for start := 0; start < len(qs); {
		end := start
		for end < len(qs) && qs[end].part == qs[start].part {
			end++
		}
		p := ExamPart{Name: qs[start].part, Case: qs[start].part != firstPart, QuestionIDs: []int64{}}
		for _, q := range qs[start:end] {
			switch {
			case !q.reserve && q.eligible:
				p.QuestionIDs = append(p.QuestionIDs, q.id)
			case !q.reserve:
				p.Missing++
				if q.annulled {
					p.Annulled++
				} else {
					p.Unpublished++
				}
			case q.eligible && p.Missing > 0:
				p.QuestionIDs = append(p.QuestionIDs, q.id)
				p.Missing--
				p.Replaced++
			}
		}
		if len(p.QuestionIDs) > 0 {
			parts = append(parts, p)
		}
		start = end
	}
	return parts
}
