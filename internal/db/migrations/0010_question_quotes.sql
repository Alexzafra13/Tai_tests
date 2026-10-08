-- The literal sentence of an official page that backs the answer, shown
-- with the link (data/bank, "articles" with url and quote).
ALTER TABLE question_sections ADD COLUMN quote TEXT NOT NULL DEFAULT '';
