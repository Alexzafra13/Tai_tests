// Study texts of the laws (internal/content/laws.go).

export type LawSectionKind = "heading" | "article" | "text";

export type LawChange = {
  date: string;
  title: string;
  body: string;
};

export type QuestionBrief = {
  id: number;
  source_ref: string;
  stem: string;
  options: [string, string, string, string];
  correct: number;
};

export type LawTextSection = {
  id: string;
  kind: LawSectionKind;
  level?: number;
  title: string;
  body: string;
  notes?: string;
  upcoming?: LawChange;
  questions: QuestionBrief[];
};

export type LawText = {
  source_id: number;
  title: string;
  reference: string;
  url: string;
  version_date: string;
  sections: LawTextSection[];
  questions: QuestionBrief[];
};

export type StudyLaw = {
  source_id: number;
  title: string;
  reference: string;
  url: string;
  version_date: string;
  parts: string[];
  cited_articles: number;
};

export type StudyTopic = {
  topic_id: number;
  code: string;
  number: number;
  title: string;
  block: string;
  laws: StudyLaw[];
};
