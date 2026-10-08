package content

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// SyllabusFile is the format of data/syllabus.json, filled in by hand from
// the official call published in the BOE.
type SyllabusFile struct {
	Name      string          `json:"name"`
	Reference string          `json:"reference"`
	Blocks    []SyllabusBlock `json:"blocks"`
}

type SyllabusBlock struct {
	Code   string          `json:"code"`
	Name   string          `json:"name"`
	Topics []SyllabusTopic `json:"topics"`
}

type SyllabusTopic struct {
	Code   string `json:"code"`
	Number int    `json:"number"`
	Title  string `json:"title"`
}

// ParseSyllabus decodes and validates a syllabus file.
func ParseSyllabus(r io.Reader) (SyllabusFile, error) {
	var f SyllabusFile
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return f, fmt.Errorf("syllabus: %w", err)
	}
	return f, f.validate()
}

func (f SyllabusFile) validate() error {
	if len(f.Blocks) == 0 {
		return fmt.Errorf("syllabus: no blocks")
	}
	codes := map[string]bool{}
	claim := func(code, what string) error {
		if strings.TrimSpace(code) == "" {
			return fmt.Errorf("syllabus: %s without code", what)
		}
		if codes[code] {
			return fmt.Errorf("syllabus: duplicate code %q", code)
		}
		codes[code] = true
		return nil
	}
	for _, b := range f.Blocks {
		if err := claim(b.Code, "block"); err != nil {
			return err
		}
		if strings.TrimSpace(b.Name) == "" {
			return fmt.Errorf("syllabus: block %s has no name", b.Code)
		}
		if len(b.Topics) == 0 {
			return fmt.Errorf("syllabus: block %s has no topics", b.Code)
		}
		for _, t := range b.Topics {
			if err := claim(t.Code, "topic in block "+b.Code); err != nil {
				return err
			}
			if strings.TrimSpace(t.Title) == "" {
				return fmt.Errorf("syllabus: topic %s has no title", t.Code)
			}
		}
	}
	return nil
}

type LoadResult struct {
	Blocks      int
	Topics      int
	Deactivated int
}

// LoadSyllabus upserts blocks and topics by code in one transaction. Blocks
// and topics missing from the file are deactivated rather than deleted so
// questions keep their topic links when the syllabus changes between calls.
func (s *Store) LoadSyllabus(ctx context.Context, f SyllabusFile) (LoadResult, error) {
	var res LoadResult
	if err := f.validate(); err != nil {
		return res, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return res, err
	}
	defer tx.Rollback()

	for _, stmt := range []string{`UPDATE blocks SET active = 0`, `UPDATE topics SET active = 0`} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return res, err
		}
	}

	for bi, b := range f.Blocks {
		var blockID int64
		err := tx.QueryRowContext(ctx, `
			INSERT INTO blocks (code, name, position, active) VALUES (?, ?, ?, 1)
			ON CONFLICT (code) DO UPDATE SET name = excluded.name, position = excluded.position, active = 1
			RETURNING id`, b.Code, strings.TrimSpace(b.Name), bi).Scan(&blockID)
		if err != nil {
			return res, fmt.Errorf("block %s: %w", b.Code, err)
		}
		res.Blocks++
		for ti, t := range b.Topics {
			number := t.Number
			if number == 0 {
				number = ti + 1
			}
			_, err := tx.ExecContext(ctx, `
				INSERT INTO topics (block_id, code, number, title, position, active) VALUES (?, ?, ?, ?, ?, 1)
				ON CONFLICT (code) DO UPDATE SET block_id = excluded.block_id, number = excluded.number,
					title = excluded.title, position = excluded.position, active = 1`,
				blockID, t.Code, number, strings.TrimSpace(t.Title), ti)
			if err != nil {
				return res, fmt.Errorf("topic %s: %w", t.Code, err)
			}
			res.Topics++
		}
	}

	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM topics WHERE active = 0`).Scan(&res.Deactivated); err != nil {
		return res, err
	}
	return res, tx.Commit()
}

type Block struct {
	ID     int64   `json:"id"`
	Code   string  `json:"code"`
	Name   string  `json:"name"`
	Topics []Topic `json:"topics"`
}

type Topic struct {
	ID        int64  `json:"id"`
	Code      string `json:"code"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Questions int    `json:"questions"`
	Published int    `json:"published"`
	// Laws counts the study texts linked to the topic.
	Laws int `json:"laws"`
	// Pages counts the official pages outside the laws that answer its
	// published questions.
	Pages int `json:"pages"`
}

// Syllabus returns the active blocks and topics in order, with question
// counts per topic.
func (s *Store) Syllabus(ctx context.Context) ([]Block, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT b.id, b.code, b.name, t.id, t.code, t.number, t.title,
			count(q.id), count(CASE WHEN q.status = 'published' AND q.annulled = 0 THEN 1 END),
			(SELECT count(*) FROM topic_laws tl WHERE tl.topic_id = t.id),
			(SELECT count(DISTINCT qs.url) FROM question_sections qs
				JOIN question_topics pt ON pt.question_id = qs.question_id AND pt.topic_id = t.id
				JOIN questions pq ON pq.id = qs.question_id AND pq.status = 'published' AND pq.annulled = 0
				WHERE qs.url <> '')
		FROM blocks b
		JOIN topics t ON t.block_id = b.id AND t.active = 1
		LEFT JOIN question_topics qt ON qt.topic_id = t.id
		LEFT JOIN questions q ON q.id = qt.question_id AND q.status <> 'discarded'
		WHERE b.active = 1
		GROUP BY t.id
		ORDER BY b.position, t.position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocks := []Block{}
	for rows.Next() {
		var b Block
		var t Topic
		if err := rows.Scan(&b.ID, &b.Code, &b.Name, &t.ID, &t.Code, &t.Number, &t.Title, &t.Questions, &t.Published, &t.Laws,
			&t.Pages); err != nil {
			return nil, err
		}
		if n := len(blocks); n == 0 || blocks[n-1].ID != b.ID {
			blocks = append(blocks, b)
		}
		last := &blocks[len(blocks)-1]
		last.Topics = append(last.Topics, t)
	}
	return blocks, rows.Err()
}
