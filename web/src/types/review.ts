// Review queue (internal/content/review.go, reports.go).

import type { Question, Status } from "./content";

export type ReviewKind = "" | "reported" | "drafts";

export type Excerpt = {
  before: string;
  match: string;
  after: string;
  clipped_start: boolean;
  clipped_end: boolean;
};

// A user's doubt about a question, raised during a test.
export type Report = { id: number; user_id: number; username: string; note: string; created_at: string };

export type ReviewItem = Question & { reports: Report[]; excerpt: Excerpt | null };

export type ReviewCounts = { reported: number; drafts: number; total: number };

export type ReviewState = { status: Status; resolved_reports: number[] };

export type BatchResult = { id: number; ok: boolean; errors?: Record<string, string> };
