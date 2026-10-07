-- Sections of the study texts that answer each official question, as the
-- bundled bank states them (data/bank, "articles"), by law reference and
-- block id so they survive reloading the laws. An empty block_id points
-- to the law as a whole. Rebuilt from the bank on every start.
CREATE TABLE question_sections (
    question_id   INTEGER NOT NULL REFERENCES questions (id) ON DELETE CASCADE,
    position      INTEGER NOT NULL,
    law_reference TEXT NOT NULL,
    block_id      TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (question_id, position)
) WITHOUT ROWID;
