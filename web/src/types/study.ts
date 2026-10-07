// Spaced repetition, statistics and search (internal/srs, internal/stats,
// content search).

import type { Origin, SourceKind } from "./content";
import type { ArticleLink } from "./laws";

export type ReviewSummary = {
  due: number;
  tracked: number;
  // mastered: next review three weeks away or more.
  mastered: number;
};

export type StudySummary = {
  review: ReviewSummary;
  failed: number;
};

export type Overview = {
  answered: number;
  correct: number;
  wrong: number;
  blank: number;
  tests_finished: number;
  study_days: number;
};

export type TopicStats = {
  block_id: number;
  block_name: string;
  topic_id: number;
  number: number;
  title: string;
  answered: number;
  correct: number;
  available: number;
  official: number;
};

export type StatsDay = {
  date: string; // YYYY-MM-DD (UTC)
  answered: number;
  correct: number;
};

export type Stats = StudySummary & {
  overview: Overview;
  topics: TopicStats[];
  timeline: StatsDay[];
};

export type SearchHit = {
  id: number;
  stem: string;
  options: [string, string, string, string];
  correct: number;
  explanation: string;
  origin: Origin;
  source_title: string;
  source_ref: string;
  source_quote: string;
  source_kind: SourceKind;
  source_url: string;
  articles: ArticleLink[];
};

export type TopicRef = {
  id: number;
  number: number;
  title: string;
};

export type StudyQuestion = SearchHit & {
  topics: TopicRef[];
};
