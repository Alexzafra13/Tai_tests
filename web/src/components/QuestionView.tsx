import { useState } from "react";
import { Link } from "react-router";
import { api, errorMessage } from "../api";
import { articlePath, optionLetters, originLabel, type ArticleLink, type Solution, type SourceKind, type TestItem } from "../types";
import "./QuestionView.css";

// Options renders the four answers as large tap targets. Once `correct` is
// known it marks the correct option and a wrong choice; before that, only
// the selection.
export function Options({
  options,
  chosen = null,
  correct,
  disabled,
  onChoose,
}: {
  options: readonly string[];
  chosen?: number | null;
  correct?: number;
  disabled?: boolean;
  onChoose?: (i: number) => void;
}) {
  return (
    <ol className="options">
      {options.map((text, i) => {
        let cls = "option";
        if (correct !== undefined) {
          if (i === correct) cls += " correct";
          else if (i === chosen) cls += " wrong";
        } else if (i === chosen) {
          cls += " selected";
        }
        return (
          <li key={i}>
            <button type="button" className={cls} disabled={disabled || !onChoose} onClick={() => onChoose?.(i)}>
              <span className="letter">{optionLetters[i]}</span>
              <span className="text">{text}</span>
            </button>
          </li>
        );
      })}
    </ol>
  );
}

const sourceLinkLabel: Record<SourceKind, string> = {
  inap_exam: "Ver el examen en el INAP",
  law: "Ver la ley en el BOE",
  technical_doc: "Ver la documentación",
};

// SourceLink opens the original document a question cites, when its URL is
// known.
export function SourceLink({ kind, url }: { kind: SourceKind; url: string }) {
  if (!url) return null;
  return (
    <a href={url} target="_blank" rel="noreferrer" className="small">
      {sourceLinkLabel[kind]} ↗
    </a>
  );
}

export function SolutionBox({ solution, chosen }: { solution: Solution; chosen: number | null }) {
  const verdict =
    solution.is_correct === null ? (
      <strong className="muted">En blanco · correcta: {optionLetters[solution.correct]}</strong>
    ) : solution.is_correct ? (
      <strong className="ok">✓ Correcta</strong>
    ) : (
      <strong className="error">
        ✗ Incorrecta · marcaste {chosen !== null && optionLetters[chosen]}, era {optionLetters[solution.correct]}
      </strong>
    );

  return (
    <div className="solution">
      {verdict}
      {solution.explanation && <p>{solution.explanation}</p>}
      <div className="source">
        <span className="muted small">
          {originLabel[solution.origin]} · {solution.source_title} · {solution.source_ref}
        </span>
        {solution.source_quote && <blockquote>{solution.source_quote}</blockquote>}
      </div>
      <ArticleLinks articles={solution.articles} />
    </div>
  );
}

// ArticleLinks opens the law at each article the question asks about, or
// the official page that answers it when no law does.
export function ArticleLinks({ articles }: { articles: ArticleLink[] }) {
  if (articles.length === 0) return null;
  return (
    <ul className="article-links">
      {articles.map((a) =>
        a.url ? (
          <li key={a.url}>
            <a href={a.url} target="_blank" rel="noreferrer" className="link">
              {a.law}
            </a>
          </li>
        ) : (
          <li key={`${a.source_id}-${a.block_id}`}>
            <Link to={articlePath(a)} className="link">
              {a.title.replace(/\.$/, "")} · {a.law}
            </Link>
          </li>
        ),
      )}
    </ul>
  );
}

// ReportControl lets the user flag a question as doubtful with an optional
// note. The report reaches the administrators' review queue.
export function ReportControl({
  testId,
  item,
  onChange,
}: {
  testId: number;
  item: TestItem;
  onChange: (reported: boolean, note: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [note, setNote] = useState(item.report_note ?? "");
  const [error, setError] = useState<string | null>(null);

  async function save(reported: boolean) {
    setError(null);
    try {
      await api(`/tests/${testId}/report`, { method: "POST", body: { position: item.position, reported, note } });
      onChange(reported, reported ? note : "");
      setOpen(false);
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  if (!open) {
    return (
      <button type="button" className={item.reported ? "flag active" : "flag"} onClick={() => setOpen(true)}>
        {item.reported ? "⚑ Dudosa" : "⚐ Marcar dudosa"}
      </button>
    );
  }
  return (
    <div className="flag-editor">
      <textarea
        rows={2}
        autoFocus
        placeholder="¿Qué te parece dudoso? (opcional)"
        value={note}
        onChange={(e) => setNote(e.target.value)}
      />
      {error && <div className="error">{error}</div>}
      <div className="actions">
        <button type="button" className="primary" onClick={() => save(true)}>
          Marcar
        </button>
        {item.reported && (
          <button type="button" onClick={() => save(false)}>
            Quitar marca
          </button>
        )}
        <button type="button" className="link" onClick={() => setOpen(false)}>
          Cancelar
        </button>
      </div>
    </div>
  );
}
