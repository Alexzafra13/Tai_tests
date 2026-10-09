package content

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/alexzafra13/tai_tests/internal/textmatch"
	"github.com/alexzafra13/tai_tests/internal/validate"
)

// Study notes (data/notes) explain a topic point by point. Nothing in them
// is made up: every point carries literal quotes of what backs it, an
// article of a bundled law or an official page, and the quotes of laws are
// checked against their text when loading. A note that fails the check is
// not loaded.

type NoteFile struct {
	Topic    string        `json:"topic"` // syllabus code, B1-T01
	Sections []NoteSection `json:"sections"`
}

type NoteSection struct {
	Title  string      `json:"title"`
	Points []NotePoint `json:"points"`
}

// NotePoint is a statement with what backs it; Items are its sub-points.
// Text may mark key words with **bold**.
type NotePoint struct {
	Text  string      `json:"text"`
	Refs  []NoteRef   `json:"refs"`
	Items []NotePoint `json:"items,omitempty"`
}

// NoteRef is a literal quote of a section of a bundled law (Law, Section)
// or of an official page (Title, URL).
type NoteRef struct {
	Law     string `json:"law,omitempty"`
	Section string `json:"section,omitempty"`
	Title   string `json:"title,omitempty"`
	URL     string `json:"url,omitempty"`
	Quote   string `json:"quote"`
}

// ReadNotes reads every *.json file at the root of fsys.
func ReadNotes(fsys fs.FS) ([]NoteFile, error) {
	names, err := fs.Glob(fsys, "*.json")
	if err != nil {
		return nil, err
	}
	var out []NoteFile
	seen := map[string]bool{}
	for _, name := range names {
		b, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, err
		}
		var n NoteFile
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&n); err != nil {
			return nil, fmt.Errorf("notes %s: %w", name, err)
		}
		if err := n.check(); err != nil {
			return nil, fmt.Errorf("notes %s: %w", name, err)
		}
		if seen[n.Topic] {
			return nil, fmt.Errorf("notes %s: topic %s has another note", name, n.Topic)
		}
		seen[n.Topic] = true
		out = append(out, n)
	}
	return out, nil
}

// MinNoteQuote is the shortest quote that can back a point.
const MinNoteQuote = 15

// check validates the shape of a note: titles, texts and a quote for every
// point.
func (n NoteFile) check() error {
	if n.Topic == "" || len(n.Sections) == 0 {
		return errors.New("topic and sections are required")
	}
	var point func(path string, p NotePoint, depth int) error
	point = func(path string, p NotePoint, depth int) error {
		if strings.TrimSpace(p.Text) == "" {
			return fmt.Errorf("point %s: empty text", path)
		}
		if len(p.Refs) == 0 {
			return fmt.Errorf("point %s: nothing backs it", path)
		}
		for _, r := range p.Refs {
			law := r.Law != "" && r.Section != "" && r.Title == "" && r.URL == ""
			page := r.Law == "" && r.Section == "" && r.Title != "" && strings.HasPrefix(r.URL, "https://")
			if !law && !page {
				return fmt.Errorf("point %s: a ref is a law and section, or a title and https url", path)
			}
			if utf8.RuneCountInString(textmatch.Normalize(r.Quote)) < MinNoteQuote {
				return fmt.Errorf("point %s: quote too short", path)
			}
		}
		if depth > 0 && len(p.Items) > 0 {
			return fmt.Errorf("point %s: sub-points cannot have sub-points", path)
		}
		for i, it := range p.Items {
			if err := point(path+"."+strconv.Itoa(i+1), it, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	for i, s := range n.Sections {
		if strings.TrimSpace(s.Title) == "" || len(s.Points) == 0 {
			return fmt.Errorf("section %d: title and points are required", i+1)
		}
		for j, p := range s.Points {
			if err := point(fmt.Sprintf("%d.%d", i+1, j+1), p, 0); err != nil {
				return err
			}
		}
	}
	return nil
}

type NoteLoadResult struct {
	Loaded   int
	Problems []string
}

// LoadNotes replaces the installation's notes with the bundled ones whose
// quotes of laws are found in the laws' text in force. Run it after
// LoadLaws.
func (s *Store) LoadNotes(ctx context.Context, notes []NoteFile) (NoteLoadResult, error) {
	var res NoteLoadResult
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		topics, err := topicsByCode(ctx, tx)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM topic_notes`); err != nil {
			return err
		}
		now := s.timestamp()
		for _, n := range notes {
			tid, ok := topics[n.Topic]
			if !ok {
				res.Problems = append(res.Problems, fmt.Sprintf("%s: unknown topic", n.Topic))
				continue
			}
			if problem, err := noteQuotesFound(ctx, tx, n); err != nil {
				return err
			} else if problem != "" {
				res.Problems = append(res.Problems, n.Topic+": "+problem)
				continue
			}
			body, err := json.Marshal(n.Sections)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO topic_notes (topic_id, body, updated_at) VALUES (?, ?, ?)`,
				tid, body, now); err != nil {
				return err
			}
			res.Loaded++
		}
		return nil
	})
	return res, err
}

// noteQuotesFound returns, for the first quote of a law that is not in the
// section it names (title, text in force or upcoming wording), what is
// wrong.
func noteQuotesFound(ctx context.Context, tx *sql.Tx, n NoteFile) (string, error) {
	var problem string
	var walk func(path string, p NotePoint) error
	walk = func(path string, p NotePoint) error {
		for _, r := range p.Refs {
			if r.Law == "" || problem != "" {
				continue
			}
			var title, body, upcoming string
			err := tx.QueryRowContext(ctx, `SELECT ls.title, ls.body, ls.upcoming_body FROM law_sections ls
				JOIN sources s ON s.id = ls.source_id AND s.kind = 'law' AND s.reference = ?
				WHERE ls.block_id = ?`, r.Law, r.Section).Scan(&title, &body, &upcoming)
			if errors.Is(err, sql.ErrNoRows) {
				problem = fmt.Sprintf("point %s: %s has no section %s", path, r.Law, r.Section)
				continue
			} else if err != nil {
				return err
			}
			if !textmatch.Contains(title+"\n"+body+"\n"+upcoming, r.Quote) {
				problem = fmt.Sprintf("point %s: quote not in %s %s", path, r.Law, r.Section)
			}
		}
		for i, it := range p.Items {
			if err := walk(path+"."+strconv.Itoa(i+1), it); err != nil {
				return err
			}
		}
		return nil
	}
	for i, sec := range n.Sections {
		for j, p := range sec.Points {
			if err := walk(fmt.Sprintf("%d.%d", i+1, j+1), p); err != nil {
				return "", err
			}
		}
	}
	return problem, nil
}

// TopicNote is a note ready to read: law quotes point to the article in the
// app.
type TopicNote struct {
	Sections []NoteSectionView `json:"sections"`
}

type NoteSectionView struct {
	Title  string          `json:"title"`
	Points []NotePointView `json:"points"`
}

type NotePointView struct {
	Path  string          `json:"path"` // "2.3", "2.3.1": what a report points at
	Text  string          `json:"text"`
	Refs  []NoteRefView   `json:"refs"`
	Items []NotePointView `json:"items"`
}

// NoteRefView is where a point comes from, with its quotes there: a law's
// section in the app (SourceID, BlockID, Label such as "Artículo 62 ·
// Constitución Española") or an official page (URL, Label its title).
type NoteRefView struct {
	SourceID int64    `json:"source_id,omitempty"`
	BlockID  string   `json:"block_id,omitempty"`
	URL      string   `json:"url,omitempty"`
	Label    string   `json:"label"`
	Quotes   []string `json:"quotes"`
	// Asked counts the official exam questions on the article.
	Asked int `json:"asked"`
}

// topicNote returns the topic's note, nil when it has none.
func (s *Store) topicNote(ctx context.Context, topicID int64) (*TopicNote, error) {
	var body []byte
	err := s.db.QueryRowContext(ctx, `SELECT body FROM topic_notes WHERE topic_id = ?`, topicID).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var secs []NoteSection
	if err := json.Unmarshal(body, &secs); err != nil {
		return nil, err
	}
	cites, err := s.lawCitations(ctx)
	if err != nil {
		return nil, err
	}
	type lawSec struct {
		sourceID int64
		label    string
		asked    int
	}
	cache := map[string]lawSec{}
	resolve := func(r NoteRef) (NoteRefView, error) {
		if r.URL != "" {
			return NoteRefView{URL: r.URL, Label: r.Title, Quotes: []string{r.Quote}}, nil
		}
		key := r.Law + "#" + r.Section
		ls, ok := cache[key]
		if !ok {
			var title, lawTitle string
			var kind LawSectionKind
			if err := s.db.QueryRowContext(ctx, `SELECT s.id, s.title, ls.title, ls.kind FROM law_sections ls
				JOIN sources s ON s.id = ls.source_id AND s.kind = 'law' AND s.reference = ?
				WHERE ls.block_id = ?`, r.Law, r.Section).Scan(&ls.sourceID, &lawTitle, &title, &kind); err != nil {
				return NoteRefView{}, err
			}
			ls.asked = len(cites[ls.sourceID].section(LawSection{ID: r.Section, Kind: kind, Title: title}))
			name, _, _ := strings.Cut(lawTitle, ",")
			name = strings.TrimSuffix(name, " del Parlamento Europeo y del Consejo")
			short, _, _ := strings.Cut(title, ". ")
			ls.label = strings.TrimSuffix(short, ".") + " · " + name
			cache[key] = ls
		}
		return NoteRefView{SourceID: ls.sourceID, BlockID: r.Section, Label: ls.label, Quotes: []string{r.Quote},
			Asked: ls.asked}, nil
	}
	var view func(path string, p NotePoint) (NotePointView, error)
	view = func(path string, p NotePoint) (NotePointView, error) {
		v := NotePointView{Path: path, Text: p.Text, Refs: []NoteRefView{}, Items: []NotePointView{}}
		at := map[string]int{} // quotes of the same section or page go together
		for _, r := range p.Refs {
			rv, err := resolve(r)
			if err != nil {
				return v, err
			}
			key := r.URL + "|" + r.Law + "#" + r.Section
			if i, ok := at[key]; ok {
				v.Refs[i].Quotes = append(v.Refs[i].Quotes, r.Quote)
				continue
			}
			at[key] = len(v.Refs)
			v.Refs = append(v.Refs, rv)
		}
		for i, it := range p.Items {
			iv, err := view(path+"."+strconv.Itoa(i+1), it)
			if err != nil {
				return v, err
			}
			v.Items = append(v.Items, iv)
		}
		return v, nil
	}
	note := &TopicNote{Sections: []NoteSectionView{}}
	for i, sec := range secs {
		sv := NoteSectionView{Title: sec.Title, Points: []NotePointView{}}
		for j, p := range sec.Points {
			pv, err := view(fmt.Sprintf("%d.%d", i+1, j+1), p)
			if err != nil {
				return nil, err
			}
			sv.Points = append(sv.Points, pv)
		}
		note.Sections = append(note.Sections, sv)
	}
	return note, nil
}

// NoteReport is a user's report on a point of a topic's note.
type NoteReport struct {
	ID         int64  `json:"id"`
	TopicID    int64  `json:"topic_id"`
	TopicCode  string `json:"topic_code"`
	TopicTitle string `json:"topic_title"`
	Point      string `json:"point"`
	Excerpt    string `json:"excerpt"`
	Username   string `json:"username"`
	Note       string `json:"note"`
	CreatedAt  string `json:"created_at"`
}

// ReportNote records a user's report on a point of a topic's note.
func (s *Store) ReportNote(ctx context.Context, userID, topicID int64, point, excerpt, note string) error {
	note = strings.TrimSpace(note)
	if note == "" {
		return validate.Errors{"note": "Cuenta qué está mal o qué falta"}
	}
	if utf8.RuneCountInString(note) > 2000 {
		return validate.Errors{"note": "Máximo 2000 caracteres"}
	}
	var exists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM topic_notes WHERE topic_id = ?)`, topicID).
		Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	if r := []rune(strings.TrimSpace(excerpt)); len(r) > 300 {
		excerpt = string(r[:300])
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO note_reports (topic_id, point, excerpt, user_id, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`, topicID, strings.TrimSpace(point), strings.TrimSpace(excerpt), userID, note, s.timestamp())
	return err
}

// OpenNoteReports lists the reports on notes not resolved yet, oldest first.
func (s *Store) OpenNoteReports(ctx context.Context) ([]NoteReport, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT r.id, r.topic_id, t.code, t.title, r.point, r.excerpt, u.username,
		r.note, r.created_at
		FROM note_reports r JOIN topics t ON t.id = r.topic_id JOIN users u ON u.id = r.user_id
		WHERE r.resolved_at = '' ORDER BY r.created_at, r.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []NoteReport{}
	for rows.Next() {
		var r NoteReport
		if err := rows.Scan(&r.ID, &r.TopicID, &r.TopicCode, &r.TopicTitle, &r.Point, &r.Excerpt, &r.Username,
			&r.Note, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ResolveNoteReport closes a report.
func (s *Store) ResolveNoteReport(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `UPDATE note_reports SET resolved_at = ? WHERE id = ? AND resolved_at = ''`,
		s.timestamp(), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
