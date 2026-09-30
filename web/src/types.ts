// Mirrors of the Go API types (internal/content).

export type SourceKind = "inap_exam" | "law" | "technical_doc";
export type Origin = "official" | "law" | "technical";
export type Author = "manual" | "ai" | "import";
export type Status = "draft" | "reviewed" | "published" | "discarded";

export type Topic = {
  id: number;
  code: string;
  number: number;
  title: string;
  questions: number;
  published: number;
};

export type Block = {
  id: number;
  code: string;
  name: string;
  topics: Topic[];
};

export type Source = {
  id: number;
  kind: SourceKind;
  title: string;
  reference: string;
  url: string;
  version_date: string;
  full_text?: string;
  has_text: boolean;
  questions: number;
};

export type SourceInput = Pick<Source, "kind" | "title" | "reference" | "url" | "version_date"> & {
  full_text: string;
};

export type Question = {
  id: number;
  stem: string;
  options: [string, string, string, string];
  correct: number;
  explanation: string;
  origin: Origin;
  author: Author;
  source_id: number;
  source_title: string;
  source_ref: string;
  source_quote: string;
  status: Status;
  annulled: boolean;
  fixed_order: boolean;
  flagged: boolean;
  flag_note: string;
  topic_ids: number[];
  revision: number;
};

export type QuestionInput = Omit<Question, "id" | "source_title" | "revision">;

// questionInput picks the editable fields of a question. The API rejects
// unknown fields, so read-only ones (id, revision, timestamps) must not be
// sent back.
export function questionInput(q: Question): QuestionInput {
  return {
    stem: q.stem,
    options: q.options,
    correct: q.correct,
    explanation: q.explanation,
    origin: q.origin,
    author: q.author,
    source_id: q.source_id,
    source_ref: q.source_ref,
    source_quote: q.source_quote,
    status: q.status,
    annulled: q.annulled,
    fixed_order: q.fixed_order,
    flagged: q.flagged,
    flag_note: q.flag_note,
    topic_ids: q.topic_ids,
  };
}

export type Page<T> = { items: T[]; total: number };

export const sourceKindLabel: Record<SourceKind, string> = {
  inap_exam: "Examen INAP",
  law: "Ley",
  technical_doc: "Documentación técnica",
};

export const originLabel: Record<Origin, string> = {
  official: "Oficial INAP",
  law: "Ley",
  technical: "Técnica",
};

// Each origin may only cite sources of one kind (enforced by the backend).
export const originSourceKind: Record<Origin, SourceKind> = {
  official: "inap_exam",
  law: "law",
  technical: "technical_doc",
};

export const statusLabel: Record<Status, string> = {
  draft: "Borrador",
  reviewed: "Revisada",
  published: "Publicada",
  discarded: "Descartada",
};

export const authorLabel: Record<Author, string> = {
  manual: "Manual",
  ai: "IA",
  import: "Importada",
};

export const optionLetters = ["A", "B", "C", "D"] as const;

// --- Review queue (internal/content/review.go) ---

export type ReviewKind = "" | "flagged" | "drafts";

export type Excerpt = {
  before: string;
  match: string;
  after: string;
  clipped_start: boolean;
  clipped_end: boolean;
};

export type ReviewItem = Question & { excerpt: Excerpt | null };

export type ReviewCounts = { flagged: number; drafts: number; total: number };

export type ReviewState = { status: Status; flagged: boolean; flag_note: string };

export type BatchResult = { id: number; ok: boolean; errors?: Record<string, string> };

// --- Tests (internal/quiz) ---

export type TestMode = "practice" | "exam";
export type TestStatus = "in_progress" | "finished" | "abandoned";

export type TestFilters = {
  topic_ids: number[];
  block_ids: number[];
  source_ids: number[];
  origins: Origin[];
  question_ids: number[];
};

export type CreateTestInput = {
  mode: TestMode;
  filters: TestFilters;
  count: number;
  penalty: number;
  time_limit_min: number;
};

export type Solution = {
  correct: number;
  is_correct: boolean | null;
  explanation: string;
  origin: Origin;
  source_title: string;
  source_ref: string;
  source_quote: string;
  topic_ids: number[];
};

export type TestItem = {
  position: number;
  question_id: number;
  stem: string;
  options: [string, string, string, string];
  chosen: number | null;
  flagged: boolean;
  flag_note?: string;
  time_ms: number;
  solution?: Solution;
};

export type TestResult = {
  total: number;
  correct: number;
  wrong: number;
  blank: number;
  penalty: number;
  net: number;
  ratio: number;
  score: number;
  score_no_penalty: number;
  max: number;
  pass_mark: number;
  passed: boolean;
};

export type Test = {
  id: number;
  mode: TestMode;
  status: TestStatus;
  penalty: number;
  time_limit: number;
  started_at: string;
  deadline?: string;
  finished_at?: string;
  remaining_sec?: number;
  result?: TestResult;
  items: TestItem[];
};

export type TestSummary = {
  id: number;
  mode: TestMode;
  status: TestStatus;
  started_at: string;
  finished_at?: string;
  deadline?: string;
  total: number;
  answered: number;
  correct: number;
  wrong: number;
  score: number | null;
  passed: boolean;
};

export type ScoringSettings = {
  max: number;
  pass_mark: number;
  default_penalty: number;
};

export const modeLabel: Record<TestMode, string> = {
  practice: "Práctica",
  exam: "Examen",
};

export const penaltyOptions = [
  { value: 0, label: "Sin penalización" },
  { value: 0.25, label: "−1/4 por error" },
  { value: 1 / 3, label: "−1/3 por error" },
  { value: 0.5, label: "−1/2 por error" },
];

export function penaltyLabel(p: number): string {
  return penaltyOptions.find((o) => Math.abs(o.value - p) < 1e-6)?.label ?? `−${p.toFixed(2)} por error`;
}
