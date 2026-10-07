package content

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/alexzafra13/tai_tests/internal/textmatch"
)

// Laws are the study texts of the syllabus: the consolidated text of each
// law as published by the BOE, split into its headings and articles, and
// bundled with the binary (data/laws) like the question bank. Nothing is
// summarised or rewritten: a section is the literal text of its block.

type LawFile struct {
	Reference string `json:"reference"` // BOE-A-…
	Title     string `json:"title"`
	URL       string `json:"url"`
	// VersionDate is the BOE's last update of the consolidated text.
	VersionDate string `json:"version_date"`
	// Aliases are how exam questions name the law ("Ley 39/2015",
	// "Constitución"), to find the articles they cite.
	Aliases  []string     `json:"aliases"`
	Sections []LawSection `json:"sections"`
}

type LawSectionKind string

const (
	SectionHeading LawSectionKind = "heading" // título, capítulo, sección…
	SectionArticle LawSectionKind = "article" // artículos and disposiciones
	SectionText    LawSectionKind = "text"    // preamble
)

type LawSection struct {
	ID    string         `json:"id"` // BOE block id: a21, ti, dadicional…
	Kind  LawSectionKind `json:"kind"`
	Level int            `json:"level,omitempty"` // headings: 1 título … 4 subsección
	Title string         `json:"title"`
	// Body holds the paragraphs in force, separated by blank lines. Empty
	// for a section that only exists from Upcoming.Date on.
	Body string `json:"body"`
	// Notes are the BOE's notes on the changes made to the section.
	Notes string `json:"notes,omitempty"`
	// Upcoming is a wording already published that comes into force later.
	Upcoming *LawChange `json:"upcoming,omitempty"`
}

type LawChange struct {
	Date  string `json:"date"` // YYYY-MM-DD
	Title string `json:"title"`
	Body  string `json:"body"`
}

// LawTopics maps a topic code to the laws studied in it. Parts limits a law
// to some of its headings (with everything under them); empty means all.
type LawTopics map[string][]LawTopicRef

type LawTopicRef struct {
	Law   string   `json:"law"`
	Parts []string `json:"parts,omitempty"`
}

// ReadLaws reads the laws (*.json) and topics.json at the root of fsys.
func ReadLaws(fsys fs.FS) ([]LawFile, LawTopics, error) {
	names, err := fs.Glob(fsys, "*.json")
	if err != nil {
		return nil, nil, err
	}
	var laws []LawFile
	var topics LawTopics
	for _, name := range names {
		b, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, nil, err
		}
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		if name == "topics.json" {
			err = dec.Decode(&topics)
		} else {
			var l LawFile
			if err = dec.Decode(&l); err == nil {
				err = l.validate()
			}
			laws = append(laws, l)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("laws %s: %w", name, err)
		}
	}
	refs := map[string]LawFile{}
	for _, l := range laws {
		refs[l.Reference] = l
	}
	for code, rs := range topics {
		for _, r := range rs {
			l, ok := refs[r.Law]
			if !ok {
				return nil, nil, fmt.Errorf("laws topics.json: %s cites unknown law %s", code, r.Law)
			}
			for _, p := range r.Parts {
				if !l.hasHeading(p) {
					return nil, nil, fmt.Errorf("laws topics.json: %s: %s has no heading %q", code, r.Law, p)
				}
			}
		}
	}
	return laws, topics, nil
}

func (l LawFile) validate() error {
	if l.Reference == "" || l.Title == "" || len(l.Sections) == 0 {
		return errors.New("reference, title and sections are required")
	}
	if _, err := time.Parse("2006-01-02", l.VersionDate); err != nil {
		return fmt.Errorf("version_date: %w", err)
	}
	ids := map[string]bool{}
	for _, s := range l.Sections {
		if s.ID == "" || ids[s.ID] {
			return fmt.Errorf("missing or repeated section id %q", s.ID)
		}
		ids[s.ID] = true
		switch s.Kind {
		case SectionHeading, SectionArticle, SectionText:
		default:
			return fmt.Errorf("section %s: kind %q", s.ID, s.Kind)
		}
	}
	return nil
}

func (l LawFile) hasHeading(id string) bool {
	for _, s := range l.Sections {
		if s.ID == id && (s.Kind == SectionHeading || s.Kind == SectionText) {
			return true
		}
	}
	return false
}

// PlainText is the text in force, used as the source's full text so that
// quotes can be checked against it.
func (l LawFile) PlainText() string {
	var b strings.Builder
	for _, s := range l.Sections {
		for _, part := range []string{s.Title, s.Body} {
			if part != "" {
				b.WriteString(part)
				b.WriteString("\n\n")
			}
		}
	}
	return strings.TrimSpace(b.String())
}

type LawLoadResult struct {
	Added, Updated int
	// Problems are laws whose new text was not applied, with why.
	Problems []string
}

// LoadLaws adds the bundled laws a database does not have, updates those
// whose version changed and replaces the topic links. A new text that would
// leave a question's quote without support is not applied.
func (s *Store) LoadLaws(ctx context.Context, laws []LawFile, topics LawTopics) (LawLoadResult, error) {
	var res LawLoadResult
	ids := map[string]int64{}
	for _, l := range laws {
		err := s.inTx(ctx, func(tx *sql.Tx) error {
			id, err := s.loadLaw(ctx, tx, l, &res)
			ids[l.Reference] = id
			return err
		})
		if err != nil {
			return res, fmt.Errorf("law %s: %w", l.Reference, err)
		}
	}
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		codes, err := topicsByCode(ctx, tx)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM topic_laws`); err != nil {
			return err
		}
		for code, refs := range topics {
			tid, ok := codes[code]
			if !ok {
				continue
			}
			for pos, r := range refs {
				if _, err := tx.ExecContext(ctx, `INSERT INTO topic_laws (topic_id, source_id, position, parts)
					VALUES (?, ?, ?, ?)`, tid, ids[r.Law], pos, strings.Join(r.Parts, ",")); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return res, err
}

func (s *Store) loadLaw(ctx context.Context, tx *sql.Tx, l LawFile, res *LawLoadResult) (int64, error) {
	var id int64
	var version string
	err := tx.QueryRowContext(ctx, `SELECT id, version_date FROM sources WHERE kind = 'law' AND reference = ?`,
		l.Reference).Scan(&id, &version)
	text := l.PlainText()
	now := s.timestamp()
	switch {
	case errors.Is(err, sql.ErrNoRows):
		err = tx.QueryRowContext(ctx, `INSERT INTO sources (kind, title, reference, url, version_date, full_text, created_at, updated_at)
			VALUES ('law', ?, ?, ?, ?, ?, ?, ?) RETURNING id`, l.Title, l.Reference, l.URL, l.VersionDate, text, now, now).Scan(&id)
		if err != nil {
			return 0, mapSourceErr(err)
		}
		res.Added++
	case err != nil:
		return 0, err
	case version == l.VersionDate:
		return id, nil
	default:
		broken, err := brokenQuotes(ctx, tx, id, text)
		if err != nil {
			return 0, err
		}
		if broken > 0 {
			res.Problems = append(res.Problems, fmt.Sprintf("%s: %s no se encuentra en la versión %s",
				l.Reference, pluralize(broken, "cita", "citas"), l.VersionDate))
			return id, nil
		}
		if _, err := tx.ExecContext(ctx, `UPDATE sources SET title = ?, url = ?, version_date = ?, full_text = ?, updated_at = ?
			WHERE id = ?`, l.Title, l.URL, l.VersionDate, text, now, id); err != nil {
			return 0, err
		}
		res.Updated++
	}

	for _, q := range []string{`DELETE FROM law_sections WHERE source_id = ?`, `DELETE FROM law_aliases WHERE source_id = ?`} {
		if _, err := tx.ExecContext(ctx, q, id); err != nil {
			return 0, err
		}
	}
	for pos, sec := range l.Sections {
		var up LawChange
		if sec.Upcoming != nil {
			up = *sec.Upcoming
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO law_sections (source_id, position, block_id, kind, level, title,
			body, notes, upcoming_date, upcoming_title, upcoming_body) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, pos, sec.ID, sec.Kind, sec.Level, sec.Title, sec.Body, sec.Notes, up.Date, up.Title, up.Body); err != nil {
			return 0, err
		}
	}
	for _, a := range l.Aliases {
		if _, err := tx.ExecContext(ctx, `INSERT INTO law_aliases (source_id, alias) VALUES (?, ?)`, id, a); err != nil {
			return 0, err
		}
	}
	return id, nil
}

// brokenQuotes counts the questions citing the source whose quote is not
// in text.
func brokenQuotes(ctx context.Context, tx *sql.Tx, sourceID int64, text string) (int, error) {
	rows, err := tx.QueryContext(ctx, `SELECT source_quote FROM questions WHERE source_id = ? AND source_quote <> ''`, sourceID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	broken := 0
	for rows.Next() {
		var quote string
		if err := rows.Scan(&quote); err != nil {
			return 0, err
		}
		if !textmatch.Contains(text, quote) {
			broken++
		}
	}
	return broken, rows.Err()
}

// StudyLaw is a law as listed for a topic.
type StudyLaw struct {
	SourceID    int64  `json:"source_id"`
	Title       string `json:"title"`
	Reference   string `json:"reference"`
	URL         string `json:"url"`
	VersionDate string `json:"version_date"`
	// Parts are the titles of the headings the topic covers; empty means
	// the whole law.
	Parts []string `json:"parts"`
	// CitedArticles counts the articles in the topic's part that official
	// questions cite.
	CitedArticles int `json:"cited_articles"`
}

type StudyTopic struct {
	TopicID int64      `json:"topic_id"`
	Code    string     `json:"code"`
	Number  int        `json:"number"`
	Title   string     `json:"title"`
	Block   string     `json:"block"`
	Laws    []StudyLaw `json:"laws"`
}

// StudyTopic returns a topic with its laws.
func (s *Store) StudyTopic(ctx context.Context, topicID int64, today time.Time) (StudyTopic, error) {
	st := StudyTopic{TopicID: topicID, Laws: []StudyLaw{}}
	err := s.db.QueryRowContext(ctx, `SELECT t.code, t.number, t.title, b.name FROM topics t
		JOIN blocks b ON b.id = t.block_id WHERE t.id = ?`, topicID).Scan(&st.Code, &st.Number, &st.Title, &st.Block)
	if errors.Is(err, sql.ErrNoRows) {
		return st, ErrNotFound
	} else if err != nil {
		return st, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT s.id, s.title, s.reference, s.url, s.version_date, tl.parts
		FROM topic_laws tl JOIN sources s ON s.id = tl.source_id WHERE tl.topic_id = ? ORDER BY tl.position`, topicID)
	if err != nil {
		return st, err
	}
	var parts []string
	for rows.Next() {
		var l StudyLaw
		var p string
		if err := rows.Scan(&l.SourceID, &l.Title, &l.Reference, &l.URL, &l.VersionDate, &p); err != nil {
			rows.Close()
			return st, err
		}
		st.Laws = append(st.Laws, l)
		parts = append(parts, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return st, err
	}
	for i := range st.Laws {
		text, err := s.LawText(ctx, st.Laws[i].SourceID, topicID, today)
		if err != nil {
			return st, err
		}
		st.Laws[i].Parts = []string{}
		if parts[i] != "" {
			for _, sec := range text.Sections {
				if sec.Kind == SectionHeading && sec.Level == text.topLevel {
					st.Laws[i].Parts = append(st.Laws[i].Parts, sec.Title)
				}
			}
		}
		for _, sec := range text.Sections {
			if len(sec.Questions) > 0 {
				st.Laws[i].CitedArticles++
			}
		}
	}
	return st, nil
}

// QuestionBrief is an official question shown next to the article it cites.
type QuestionBrief struct {
	ID        int64     `json:"id"`
	SourceRef string    `json:"source_ref"`
	Stem      string    `json:"stem"`
	Options   [4]string `json:"options"`
	Correct   int       `json:"correct"`
}

type LawTextSection struct {
	LawSection
	// Questions are the published questions that cite this article.
	Questions []QuestionBrief `json:"questions"`
}

type LawText struct {
	SourceID    int64            `json:"source_id"`
	Title       string           `json:"title"`
	Reference   string           `json:"reference"`
	URL         string           `json:"url"`
	VersionDate string           `json:"version_date"`
	Sections    []LawTextSection `json:"sections"`
	// Questions are published questions about the law that cite no article.
	Questions []QuestionBrief `json:"questions"`
	topLevel  int
}

// LawText returns a law as in force on today, limited to the parts of
// topicID when it is not 0, with the official questions that cite it.
func (s *Store) LawText(ctx context.Context, sourceID, topicID int64, today time.Time) (LawText, error) {
	lt := LawText{SourceID: sourceID, Sections: []LawTextSection{}, Questions: []QuestionBrief{}}
	err := s.db.QueryRowContext(ctx, `SELECT title, reference, url, version_date FROM sources WHERE id = ? AND kind = 'law'`,
		sourceID).Scan(&lt.Title, &lt.Reference, &lt.URL, &lt.VersionDate)
	if errors.Is(err, sql.ErrNoRows) {
		return lt, ErrNotFound
	} else if err != nil {
		return lt, err
	}
	var parts string
	if topicID != 0 {
		err := s.db.QueryRowContext(ctx, `SELECT parts FROM topic_laws WHERE topic_id = ? AND source_id = ?`,
			topicID, sourceID).Scan(&parts)
		if errors.Is(err, sql.ErrNoRows) {
			return lt, ErrNotFound
		} else if err != nil {
			return lt, err
		}
	}

	rows, err := s.db.QueryContext(ctx, `SELECT block_id, kind, level, title, body, notes, upcoming_date, upcoming_title,
		upcoming_body FROM law_sections WHERE source_id = ? ORDER BY position`, sourceID)
	if err != nil {
		return lt, err
	}
	var all []LawSection
	for rows.Next() {
		var sec LawSection
		var up LawChange
		if err := rows.Scan(&sec.ID, &sec.Kind, &sec.Level, &sec.Title, &sec.Body, &sec.Notes, &up.Date, &up.Title,
			&up.Body); err != nil {
			rows.Close()
			return lt, err
		}
		if up.Date != "" {
			sec.Upcoming = &up
		}
		all = append(all, sec.inForce(today))
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return lt, err
	}

	keep := selectParts(all, parts)
	byArticle, general, err := s.lawQuestions(ctx, sourceID)
	if err != nil {
		return lt, err
	}
	for i, sec := range all {
		if !keep[i] || (sec.Body == "" && sec.Upcoming == nil && sec.Kind != SectionHeading) {
			continue
		}
		if sec.Kind == SectionHeading && (lt.topLevel == 0 || sec.Level < lt.topLevel) {
			lt.topLevel = sec.Level
		}
		ts := LawTextSection{LawSection: sec, Questions: []QuestionBrief{}}
		if sec.Kind == SectionArticle {
			if qs := byArticle[articleKey(sec.Title)]; qs != nil {
				ts.Questions = qs
			}
		}
		lt.Sections = append(lt.Sections, ts)
	}
	if topicID == 0 || parts == "" {
		lt.Questions = general
	}
	return lt, nil
}

// inForce applies the upcoming wording once its date has come.
func (sec LawSection) inForce(today time.Time) LawSection {
	if sec.Upcoming != nil && sec.Upcoming.Date <= today.Format("2006-01-02") {
		if sec.Upcoming.Title != "" {
			sec.Title = sec.Upcoming.Title
		}
		sec.Body = sec.Upcoming.Body
		sec.Upcoming = nil
	}
	return sec
}

// selectParts marks the sections under the given headings (comma
// separated block ids): each heading with everything until the next heading
// of the same or a higher level. Empty parts select everything.
func selectParts(all []LawSection, parts string) []bool {
	keep := make([]bool, len(all))
	if parts == "" {
		for i := range keep {
			keep[i] = true
		}
		return keep
	}
	want := map[string]bool{}
	for _, p := range strings.Split(parts, ",") {
		want[p] = true
	}
	in, level := false, 0
	for i, sec := range all {
		if sec.Kind == SectionHeading || sec.Kind == SectionText {
			lvl := sec.Level
			if sec.Kind == SectionText {
				lvl = 1
			}
			if in && lvl <= level {
				in = false
			}
			if !in && want[sec.ID] {
				in, level = true, lvl
			}
		}
		keep[i] = in
	}
	return keep
}

var articleTitle = regexp.MustCompile(`^Artículo (\d+)(?: (bis|ter|quater))?\b`)

// articleKey identifies an article by its number ("21", "4bis") from its
// title; BOE block ids do not follow the numbering ("a4-2" is 4 bis).
func articleKey(title string) string {
	m := articleTitle.FindStringSubmatch(title)
	if m == nil {
		return ""
	}
	return m[1] + m[2]
}

var articleRange = regexp.MustCompile(`^\s*(a|al)\s+(el\s+)?\d+`)

var articleRef = regexp.MustCompile(`(?i)\bart(?:[íi]culos?|\.)\s+(\d+)(?:\s*(bis|ter|quater))?\b`)

// lawQuestions finds the published official questions that name the law
// by one of its aliases. An article the stem cites ("artículo 21",
// "art. 62") is attributed to the nearest alias; questions citing no
// single article of the law go to general.
func (s *Store) lawQuestions(ctx context.Context, sourceID int64) (map[string][]QuestionBrief, []QuestionBrief, error) {
	aliases, err := s.lawAliases(ctx)
	if err != nil {
		return nil, nil, err
	}
	mine := aliases[sourceID]
	if len(mine) == 0 {
		return nil, []QuestionBrief{}, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, source_ref, stem, option_a, option_b, option_c, option_d, correct
		FROM questions WHERE status = 'published' AND annulled = 0 ORDER BY source_ref`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	byArticle := map[string][]QuestionBrief{}
	general := []QuestionBrief{}
	for rows.Next() {
		var q QuestionBrief
		if err := rows.Scan(&q.ID, &q.SourceRef, &q.Stem, &q.Options[0], &q.Options[1], &q.Options[2], &q.Options[3],
			&q.Correct); err != nil {
			return nil, nil, err
		}
		// Practical-case questions start with the case statement.
		if i := strings.LastIndex(q.Stem, "\n\n"); i >= 0 {
			q.Stem = q.Stem[i+2:]
		}
		// Wrong options name laws and articles on purpose: only the stem
		// and the right answer count.
		articles, named := citedArticles(q.Stem, q.Options[q.Correct], sourceID, aliases)
		if !named {
			continue
		}
		if len(articles) == 0 {
			general = append(general, q)
		}
		for _, a := range articles {
			byArticle[a] = append(byArticle[a], q)
		}
	}
	return byArticle, general, rows.Err()
}

// lawAliases returns, by law, patterns matching its aliases as whole words
// (case-sensitive, so "CE" is not any "ce").
func (s *Store) lawAliases(ctx context.Context) (map[int64][]*regexp.Regexp, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT source_id, alias FROM law_aliases`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64][]*regexp.Regexp{}
	for rows.Next() {
		var id int64
		var a string
		if err := rows.Scan(&id, &a); err != nil {
			return nil, err
		}
		re, err := regexp.Compile(`(^|[^\p{L}\d/])` + regexp.QuoteMeta(a) + `($|[^\p{L}\d/])`)
		if err != nil {
			return nil, err
		}
		out[id] = append(out[id], re)
	}
	return out, rows.Err()
}

// citedArticles returns the keys (see articleKey) of the articles of law
// the stem cites, and whether the stem or the answer name law at all.
// Ranges ("artículos 15 a 29") cite no single article.
func citedArticles(stem, answer string, law int64, aliases map[int64][]*regexp.Regexp) ([]string, bool) {
	text := stem
	type mention struct {
		law int64
		pos int
	}
	var mentions []mention
	for id, as := range aliases {
		for _, re := range as {
			for _, m := range re.FindAllStringIndex(text, -1) {
				mentions = append(mentions, mention{id, m[0]})
			}
		}
	}
	named := false
	for _, m := range mentions {
		named = named || m.law == law
	}
	for _, re := range aliases[law] {
		named = named || re.MatchString(answer)
	}
	if !named {
		return nil, false
	}
	seen := map[string]bool{}
	var out []string
	for _, m := range articleRef.FindAllStringSubmatchIndex(text, -1) {
		if articleRange.MatchString(text[m[1]:]) {
			continue
		}
		best, dist := int64(0), -1
		for _, mn := range mentions {
			d := mn.pos - m[0]
			if d < 0 {
				d = -d
			}
			if dist < 0 || d < dist {
				best, dist = mn.law, d
			}
		}
		if best != law {
			continue
		}
		id := text[m[2]:m[3]]
		if m[4] >= 0 {
			id += strings.ToLower(text[m[4]:m[5]])
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out, true
}
