import { useState } from "react";
import { api, errorMessage } from "../api";
import { optionLetters, originLabel, type Solution, type TestItem } from "../types";

// Options renders the four answers as large tap targets. With a solution it
// colours the correct option and a wrong choice; otherwise it shows the
// current selection.
export function Options({
  item,
  disabled,
  onChoose,
}: {
  item: TestItem;
  disabled?: boolean;
  onChoose?: (i: number) => void;
}) {
  const sol = item.solution;
  return (
    <ol className="options">
      {item.options.map((text, i) => {
        let cls = "option";
        if (sol) {
          if (i === sol.correct) cls += " correct";
          else if (i === item.chosen) cls += " wrong";
        } else if (i === item.chosen) {
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
