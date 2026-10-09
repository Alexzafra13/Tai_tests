-- Whether someone changed the question in this installation (editing it
-- or deciding on it in Review). The bank only brings its corrections to
-- questions nobody has touched. Timestamps cannot tell: they have
-- millisecond resolution.
ALTER TABLE questions ADD COLUMN edited INTEGER NOT NULL DEFAULT 0 CHECK (edited IN (0, 1));
UPDATE questions SET edited = 1 WHERE updated_at <> created_at;
