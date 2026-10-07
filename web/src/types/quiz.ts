// Tests, attempts and scoring (internal/quiz).

import type { Origin } from "./content";

export type TestMode = "practice" | "exam";
export type TestStatus = "in_progress" | "finished" | "abandoned";

export type TestFilters = {
  topic_ids: number[];
  block_ids: number[];
  source_ids: number[];
  origins: Origin[];
  question_ids: number[];
  // ordered: keep the order of question_ids instead of shuffling.
  ordered?: boolean;
  // due: only questions due for spaced-repetition review today.
  due?: boolean;
  // failed: only questions whose last answer was wrong.
  failed?: boolean;
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
  // reported: the current user has an open doubt about this question.
  reported: boolean;
  report_note?: string;
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

// An official exam split as the candidate takes it (internal/content/exams.go).
export type ExamPart = {
  name: string;
  // case: a practical case; only one of them is answered.
  case: boolean;
  question_ids: number[];
  // replaced: questions left out (annulled, unpublished) and taken over by reserves.
  replaced: number;
  // missing: those left out with no reserve to take over.
  missing: number;
};

export type Exam = {
  id: number;
  title: string;
  reference: string;
  url: string;
  parts: ExamPart[];
};
