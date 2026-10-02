import { useState } from "react";
import { api, errorMessage } from "../api";
import { optionLetters, originLabel, type Solution, type TestItem } from "../types";
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
    </div>
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
