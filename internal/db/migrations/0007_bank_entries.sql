-- Questions loaded from the bank bundled in the binary (data/bank), by
-- source and key. Rows stay when the question is edited, discarded or
-- deleted, so reloading the bank never brings it back.
CREATE TABLE bank_entries (
    source_kind      TEXT NOT NULL,
    source_reference TEXT NOT NULL,
    question_key     TEXT NOT NULL,
    loaded_at        TEXT NOT NULL,
    PRIMARY KEY (source_kind, source_reference, question_key)
) WITHOUT ROWID;
