-- Study notes of each topic, bundled with the binary (data/notes) and
-- rebuilt from it on every start. body is the note as JSON: sections of
-- points, each backed by literal quotes of the study texts.
CREATE TABLE topic_notes (
    topic_id   INTEGER PRIMARY KEY REFERENCES topics (id),
    body       TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Users' reports on a point of a note ("this is wrong", "this is missing").
-- point is the point's path in the note (e.g. "2.3"); a report stays open
-- until an administrator marks it resolved.
CREATE TABLE note_reports (
    id          INTEGER PRIMARY KEY,
    topic_id    INTEGER NOT NULL REFERENCES topics (id),
    point       TEXT NOT NULL,
    excerpt     TEXT NOT NULL DEFAULT '',
    user_id     INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    note        TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    resolved_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX note_reports_open ON note_reports (resolved_at, created_at);
