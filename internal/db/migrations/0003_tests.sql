-- migrate:rebuild
-- Rebuild questions to add the 'discarded' status (questions with answer
-- history are discarded instead of deleted) and a revision counter that
-- attempts record, so stats can tell which version of a question was seen.

CREATE TABLE questions_new (
    id           INTEGER PRIMARY KEY,
    stem         TEXT NOT NULL CHECK (trim(stem) <> ''),
    option_a     TEXT NOT NULL CHECK (trim(option_a) <> ''),
    option_b     TEXT NOT NULL CHECK (trim(option_b) <> ''),
    option_c     TEXT NOT NULL CHECK (trim(option_c) <> ''),
    option_d     TEXT NOT NULL CHECK (trim(option_d) <> ''),
    correct      INTEGER NOT NULL CHECK (correct BETWEEN 0 AND 3),
    explanation  TEXT NOT NULL DEFAULT '',
    origin       TEXT NOT NULL CHECK (origin IN ('official', 'law', 'technical')),
    author       TEXT NOT NULL CHECK (author IN ('manual', 'ai', 'import')),
    source_id    INTEGER NOT NULL REFERENCES sources (id),
    source_ref   TEXT NOT NULL CHECK (trim(source_ref) <> ''),
    source_quote TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'reviewed', 'published', 'discarded')),
    annulled     INTEGER NOT NULL DEFAULT 0 CHECK (annulled IN (0, 1)),
    flagged      INTEGER NOT NULL DEFAULT 0 CHECK (flagged IN (0, 1)),
    flag_note    TEXT NOT NULL DEFAULT '',
    revision     INTEGER NOT NULL DEFAULT 1,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    CHECK (origin = 'official' OR trim(source_quote) <> '')
);

INSERT INTO questions_new (id, stem, option_a, option_b, option_c, option_d, correct, explanation, origin,
    author, source_id, source_ref, source_quote, status, annulled, flagged, flag_note, created_at, updated_at)
SELECT id, stem, option_a, option_b, option_c, option_d, correct, explanation, origin,
    author, source_id, source_ref, source_quote, status, annulled, flagged, flag_note, created_at, updated_at
FROM questions;

DROP TABLE questions;
ALTER TABLE questions_new RENAME TO questions;

CREATE INDEX questions_status ON questions (status, annulled);
CREATE INDEX questions_source ON questions (source_id);

-- A test session. The question set and order are fixed at creation so the
-- test can be resumed after closing the app.
CREATE TABLE tests (
    id          INTEGER PRIMARY KEY,
    mode        TEXT NOT NULL CHECK (mode IN ('practice', 'exam')),
    config      TEXT NOT NULL DEFAULT '{}', -- filters used to build it (JSON)
    penalty     REAL NOT NULL CHECK (penalty >= 0 AND penalty <= 1),
    time_limit  INTEGER NOT NULL DEFAULT 0 CHECK (time_limit >= 0), -- seconds, 0 = none
    status      TEXT NOT NULL DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'finished', 'abandoned')),
    started_at  TEXT NOT NULL,
    deadline    TEXT NOT NULL DEFAULT '', -- started_at + time_limit, absolute: the clock runs while away
    finished_at TEXT NOT NULL DEFAULT '',
    correct     INTEGER NOT NULL DEFAULT 0,
    wrong       INTEGER NOT NULL DEFAULT 0,
    blank       INTEGER NOT NULL DEFAULT 0,
    score       REAL -- 0-10 after penalty, set on finish
);

CREATE INDEX tests_status ON tests (status, started_at);

-- One row per question in a test, created with the test and filled in as
-- they are answered. chosen NULL means not answered (blank once finished).
CREATE TABLE attempts (
    test_id     INTEGER NOT NULL REFERENCES tests (id),
    position    INTEGER NOT NULL,
    question_id INTEGER NOT NULL REFERENCES questions (id),
    revision    INTEGER NOT NULL,
    chosen      INTEGER CHECK (chosen BETWEEN 0 AND 3),
    is_correct  INTEGER CHECK (is_correct IN (0, 1)),
    time_ms     INTEGER NOT NULL DEFAULT 0,
    answered_at TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (test_id, position)
) WITHOUT ROWID;

CREATE INDEX attempts_question ON attempts (question_id, answered_at);
