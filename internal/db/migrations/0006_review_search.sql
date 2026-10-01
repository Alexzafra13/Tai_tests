-- Spaced repetition: one FSRS card per user and question. The card itself
-- (stability, difficulty...) is stored as JSON; due is a column so "what is
-- due today" is a plain indexed query.
CREATE TABLE review_cards (
    user_id     INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    question_id INTEGER NOT NULL REFERENCES questions (id) ON DELETE CASCADE,
    due         TEXT NOT NULL,
    card        TEXT NOT NULL,
    reps        INTEGER NOT NULL DEFAULT 0,
    lapses      INTEGER NOT NULL DEFAULT 0,
    last_rating INTEGER NOT NULL DEFAULT 0,
    updated_at  TEXT NOT NULL,
    PRIMARY KEY (user_id, question_id)
) WITHOUT ROWID;

CREATE INDEX review_cards_due ON review_cards (user_id, due);

-- Full-text search over questions, accent-insensitive ("proteccion" finds
-- "protección"). Kept in sync by triggers; rowid is the question id.
-- Note: a migration that rebuilds the questions table must recreate these
-- triggers.
CREATE VIRTUAL TABLE questions_fts USING fts5(
    stem, options, explanation, source_ref,
    tokenize = 'unicode61 remove_diacritics 2'
);

INSERT INTO questions_fts (rowid, stem, options, explanation, source_ref)
SELECT id, stem, option_a || ' ' || option_b || ' ' || option_c || ' ' || option_d, explanation, source_ref
FROM questions;

CREATE TRIGGER questions_fts_insert AFTER INSERT ON questions BEGIN
    INSERT INTO questions_fts (rowid, stem, options, explanation, source_ref)
    VALUES (new.id, new.stem, new.option_a || ' ' || new.option_b || ' ' || new.option_c || ' ' || new.option_d,
        new.explanation, new.source_ref);
END;

CREATE TRIGGER questions_fts_delete AFTER DELETE ON questions BEGIN
    DELETE FROM questions_fts WHERE rowid = old.id;
END;

CREATE TRIGGER questions_fts_update AFTER UPDATE OF stem, option_a, option_b, option_c, option_d, explanation, source_ref
ON questions BEGIN
    DELETE FROM questions_fts WHERE rowid = old.id;
    INSERT INTO questions_fts (rowid, stem, options, explanation, source_ref)
    VALUES (new.id, new.stem, new.option_a || ' ' || new.option_b || ' ' || new.option_c || ' ' || new.option_d,
        new.explanation, new.source_ref);
END;
