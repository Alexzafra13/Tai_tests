-- Official syllabus. Codes are stable identifiers taken from syllabus.json so
-- the file can be reloaded without breaking links. Topics dropped from a new
-- syllabus are deactivated, never deleted, to keep question history.
CREATE TABLE blocks (
    id       INTEGER PRIMARY KEY,
    code     TEXT NOT NULL UNIQUE,
    name     TEXT NOT NULL,
    position INTEGER NOT NULL,
    active   INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1))
);

CREATE TABLE topics (
    id       INTEGER PRIMARY KEY,
    block_id INTEGER NOT NULL REFERENCES blocks (id),
    code     TEXT NOT NULL UNIQUE,
    number   INTEGER NOT NULL,
    title    TEXT NOT NULL,
    position INTEGER NOT NULL,
    active   INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1))
);

CREATE INDEX topics_block ON topics (block_id, position);

-- Documents questions come from: official exams, BOE laws, technical docs.
CREATE TABLE sources (
    id           INTEGER PRIMARY KEY,
    kind         TEXT NOT NULL CHECK (kind IN ('inap_exam', 'law', 'technical_doc')),
    title        TEXT NOT NULL CHECK (trim(title) <> ''),
    reference    TEXT NOT NULL DEFAULT '', -- BOE-A-..., call and year, RFC number...
    url          TEXT NOT NULL DEFAULT '',
    version_date TEXT NOT NULL DEFAULT '', -- consolidated text date (YYYY-MM-DD)
    full_text    TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

CREATE UNIQUE INDEX sources_kind_reference ON sources (kind, reference) WHERE reference <> '';

CREATE TABLE questions (
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
    -- "Nothing made up": every question points at its source. Official
    -- questions are justified by the answer key; the rest need a literal quote.
    source_id    INTEGER NOT NULL REFERENCES sources (id),
    source_ref   TEXT NOT NULL CHECK (trim(source_ref) <> ''),
    source_quote TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'reviewed', 'published')),
    annulled     INTEGER NOT NULL DEFAULT 0 CHECK (annulled IN (0, 1)),
    flagged      INTEGER NOT NULL DEFAULT 0 CHECK (flagged IN (0, 1)),
    flag_note    TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    CHECK (origin = 'official' OR trim(source_quote) <> '')
);

CREATE INDEX questions_status ON questions (status, annulled);
CREATE INDEX questions_source ON questions (source_id);

CREATE TABLE question_topics (
    question_id INTEGER NOT NULL REFERENCES questions (id) ON DELETE CASCADE,
    topic_id    INTEGER NOT NULL REFERENCES topics (id),
    PRIMARY KEY (question_id, topic_id)
) WITHOUT ROWID;

CREATE INDEX question_topics_topic ON question_topics (topic_id, question_id);
