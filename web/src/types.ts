// Mirrors of the Go API types (internal/content).

export type SourceKind = "inap_exam" | "law" | "technical_doc";
export type Origin = "official" | "law" | "technical";
export type Author = "manual" | "ai" | "import";
export type Status = "draft" | "reviewed" | "published";

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
  flagged: boolean;
  flag_note: string;
  topic_ids: number[];
};

export type QuestionInput = Omit<Question, "id" | "source_title">;

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
};

export const authorLabel: Record<Author, string> = {
  manual: "Manual",
  ai: "IA",
  import: "Importada",
};

export const optionLetters = ["A", "B", "C", "D"] as const;
