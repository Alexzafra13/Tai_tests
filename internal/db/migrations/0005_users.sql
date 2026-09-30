-- migrate:rebuild
-- Multiple users with roles. Existing data (tests, doubt flags) is assigned
-- to the first administrator, created here without a password: it is set
-- from TAI_ADMIN_PASSWORD on first start.

CREATE TABLE users (
    id            INTEGER PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE COLLATE NOCASE CHECK (length(username) BETWEEN 3 AND 32),
    display_name  TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL DEFAULT '', -- bcrypt; '' means the account cannot log in yet
    role          TEXT NOT NULL CHECK (role IN ('admin', 'user')),
    active        INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);

INSERT INTO users (id, username, role, created_at, updated_at)
VALUES (1, 'admin', 'admin', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));

-- Sessions now belong to a user; old single-user sessions are dropped (log in again).
DROP TABLE sessions;
CREATE TABLE sessions (
    token_hash TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    expires_at TEXT NOT NULL
);
CREATE INDEX sessions_expires_at ON sessions (expires_at);
CREATE INDEX sessions_user ON sessions (user_id);

-- Tests belong to the user who takes them.
CREATE TABLE tests_new (
    id          INTEGER PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users (id),
    mode        TEXT NOT NULL CHECK (mode IN ('practice', 'exam')),
    config      TEXT NOT NULL DEFAULT '{}',
    penalty     REAL NOT NULL CHECK (penalty >= 0 AND penalty <= 1),
    time_limit  INTEGER NOT NULL DEFAULT 0 CHECK (time_limit >= 0),
    status      TEXT NOT NULL DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'finished', 'abandoned')),
    started_at  TEXT NOT NULL,
    deadline    TEXT NOT NULL DEFAULT '',
    finished_at TEXT NOT NULL DEFAULT '',
    correct     INTEGER NOT NULL DEFAULT 0,
    wrong       INTEGER NOT NULL DEFAULT 0,
    blank       INTEGER NOT NULL DEFAULT 0,
    score       REAL -- net ratio (net / total), set on finish
);
INSERT INTO tests_new (id, user_id, mode, config, penalty, time_limit, status, started_at, deadline, finished_at,
    correct, wrong, blank, score)
SELECT id, 1, mode, config, penalty, time_limit, status, started_at, deadline, finished_at, correct, wrong, blank, score
FROM tests;
DROP TABLE tests;
ALTER TABLE tests_new RENAME TO tests;
CREATE INDEX tests_user_status ON tests (user_id, status, started_at);

-- Doubts are reports by a user with a note, instead of a single flag on the
-- question. A report stays open until the question is reviewed.
CREATE TABLE question_reports (
    id          INTEGER PRIMARY KEY,
    question_id INTEGER NOT NULL REFERENCES questions (id) ON DELETE CASCADE,
    user_id     INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    note        TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL,
    resolved_at TEXT NOT NULL DEFAULT '' -- '' while open
);
CREATE UNIQUE INDEX question_reports_open ON question_reports (question_id, user_id) WHERE resolved_at = '';
CREATE INDEX question_reports_question ON question_reports (question_id, resolved_at);

INSERT INTO question_reports (question_id, user_id, note, created_at)
SELECT id, 1, flag_note, updated_at FROM questions WHERE flagged = 1;

ALTER TABLE questions DROP COLUMN flag_note;
ALTER TABLE questions DROP COLUMN flagged;
