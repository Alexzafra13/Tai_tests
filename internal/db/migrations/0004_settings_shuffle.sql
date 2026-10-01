-- User settings as JSON values by key (scoring scale, defaults...).
CREATE TABLE settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Options are shuffled per test unless this is set, or unless an option
-- refers to others by letter ("A y B son correctas").
ALTER TABLE questions ADD COLUMN fixed_order INTEGER NOT NULL DEFAULT 0 CHECK (fixed_order IN (0, 1));

-- Order in which the options were shown: "2031" means the first option shown
-- was the original option 2 (C). chosen always stores the original index.
ALTER TABLE attempts ADD COLUMN option_order TEXT NOT NULL DEFAULT '0123';

-- tests.score now stores the net ratio (net / total, at most 1) instead of a
-- 0-10 mark, so the display scale can be changed in the settings.
UPDATE tests SET score = score / 10 WHERE score IS NOT NULL;
