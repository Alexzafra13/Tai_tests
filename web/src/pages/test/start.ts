import { api } from "../../api";
import type { CreateTestInput, TestFilters } from "../../types";

export const emptyFilters: TestFilters = {
  topic_ids: [],
  block_ids: [],
  source_ids: [],
  origins: [],
  question_ids: [],
  due: false,
  failed: false,
};

// QUICK_COUNT is the size of one-tap review tests: short enough for a break.
export const QUICK_COUNT = 20;

// startPractice creates a practice test (no time limit) and returns its id.
export async function startPractice(filters: Partial<TestFilters>, count: number, penalty = 1 / 3): Promise<number> {
  const body: CreateTestInput = {
    mode: "practice",
    count,
    penalty,
    time_limit_min: 0,
    filters: { ...emptyFilters, ...filters },
  };
  const { id } = await api<{ id: number }>("/tests", { method: "POST", body });
  return id;
}
