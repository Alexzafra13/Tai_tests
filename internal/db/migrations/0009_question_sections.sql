-- What answers each official question, as the bundled bank states it
-- (data/bank, "articles"): a section of a study text, by law reference and
-- block id so it survives reloading the laws (an empty block_id points to
-- the law as a whole), or an official page outside the laws, by url.
-- Rebuilt from the bank on every start.
CREATE TABLE question_sections (
    question_id   INTEGER NOT NULL REFERENCES questions (id) ON DELETE CASCADE,
    position      INTEGER NOT NULL,
    law_reference TEXT NOT NULL DEFAULT '',
    block_id      TEXT NOT NULL DEFAULT '',
    title         TEXT NOT NULL DEFAULT '',
    url           TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (question_id, position)
) WITHOUT ROWID;
