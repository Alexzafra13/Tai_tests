import { useState } from "react";
import { api, errorMessage } from "../api";
import { optionLetters, originLabel, type Solution, type TestItem } from "../types";

// Options renders the four answers as large tap targets. When the correct
// option is known it is coloured, along with a wrong choice; otherwise the
// current selection is shown.
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

// FlagControl marks the question as doubtful with an optional note, sending
// it to the review queue.
export function FlagControl({
  testId,
  item,
  onChange,
}: {
  testId: number;
  item: TestItem;
  onChange: (flagged: boolean, note: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [note, setNote] = useState(item.flag_note ?? "");
  const [error, setError] = useState<string | null>(null);

  async function save(flagged: boolean) {
    setError(null);
    try {
      await api(`/tests/${testId}/flag`, { method: "POST", body: { position: item.position, flagged, note } });
      onChange(flagged, flagged ? note : "");
      setOpen(false);
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  if (!open) {
    return (
      <button type="button" className={item.flagged ? "flag active" : "flag"} onClick={() => setOpen(true)}>
        {item.flagged ? "⚑ Dudosa" : "⚐ Marcar dudosa"}
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
        {item.flagged && (
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
