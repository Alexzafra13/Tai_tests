-- Study texts: each law (a source of kind 'law') split into its BOE blocks,
-- in order. upcoming_* hold a wording already published that comes into
-- force on upcoming_date; the app shows it from that day on.
CREATE TABLE law_sections (
    source_id      INTEGER NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    position       INTEGER NOT NULL,
    block_id       TEXT NOT NULL,
    kind           TEXT NOT NULL CHECK (kind IN ('heading', 'article', 'text')),
    level          INTEGER NOT NULL DEFAULT 0,
    title          TEXT NOT NULL DEFAULT '',
    body           TEXT NOT NULL DEFAULT '',
    notes          TEXT NOT NULL DEFAULT '',
    upcoming_date  TEXT NOT NULL DEFAULT '',
    upcoming_title TEXT NOT NULL DEFAULT '',
    upcoming_body  TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (source_id, position)
) WITHOUT ROWID;

-- How exam questions name each law ("Ley 39/2015", "Constitución"), to
-- link the articles they cite.
CREATE TABLE law_aliases (
    source_id INTEGER NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    alias     TEXT NOT NULL,
    PRIMARY KEY (source_id, alias)
) WITHOUT ROWID;

-- Laws studied in each topic; parts limits a law to some of its headings
-- (comma-separated block ids, empty for the whole law).
CREATE TABLE topic_laws (
    topic_id  INTEGER NOT NULL REFERENCES topics (id),
    source_id INTEGER NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    position  INTEGER NOT NULL,
    parts     TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (topic_id, source_id)
) WITHOUT ROWID;
