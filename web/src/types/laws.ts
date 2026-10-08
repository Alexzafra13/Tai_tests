// Study texts of the laws (internal/content/laws.go).

export type LawSectionKind = "heading" | "article" | "text";

export type LawChange = {
  date: string;
  title: string;
  body: string;
};

// ArticleLink points from a question to an article it cites; topic_id is 0
// when no topic studies that article. With url, it is an official page
// outside the laws and law holds its title.
export type ArticleLink = {
  url?: string;
  source_id: number;
  law: string;
  block_id: string;
  title: string;
  topic_id: number;
};

// articlePath opens the law at the article, within its topic when it has one.
export function articlePath(a: ArticleLink): string {
  const law = a.topic_id ? `/study/${a.topic_id}/laws/${a.source_id}` : `/laws/${a.source_id}`;
  return `${law}#${a.block_id}`;
}

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
