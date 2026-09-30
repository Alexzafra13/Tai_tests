import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { api, ApiError, errorMessage } from "../../api";
import { useCountdown } from "../../hooks";
import { formatClock } from "../../format";
import { modeLabel, type Solution, type Test, type TestItem } from "../../types";
import { ErrorBox } from "../../components/Form";
import { FlagControl, Options, SolutionBox } from "../../components/QuestionView";
import { Navigator } from "./Navigator";

// Runner shows one question at a time. Every answer is saved immediately,
// so leaving and coming back resumes the test.
export function Runner({ initial, onFinished }: { initial: Test; onFinished: () => void }) {
  const navigate = useNavigate();
  const [test, setTest] = useState(initial);
  const [params, setParams] = useSearchParams();
  const [error, setError] = useState<string | null>(null);
  const [showIndex, setShowIndex] = useState(false);
  const [busy, setBusy] = useState(false);

  // Resume at the requested position, else at the first question that was
  // unanswered when the test was opened (computed once: recomputing it would
  // jump away from a question as soon as it is answered).
  const [resumeAt] = useState(() => Math.max(0, initial.items.findIndex((it) => it.chosen === null)));
  const index = Math.min(Number(params.get("q") ?? resumeAt), test.items.length - 1);
  const item = test.items[index];
  const isExam = test.mode === "exam";
  const answered = test.items.filter((it) => it.chosen !== null).length;
  const isLast = index === test.items.length - 1;
  const practiceDone = !isExam && item.solution !== undefined;

  const goTo = useCallback(
    (i: number) => {
      setParams({ q: String(i) }, { replace: true });
      setShowIndex(false);
      window.scrollTo(0, 0);
    },
    [setParams],
  );

  // Time spent on the current question, sent with the answer.
  const shownAt = useRef(Date.now());
  useEffect(() => {
    shownAt.current = Date.now();
  }, [index]);

  const remaining = useCountdown(test.remaining_sec);
  useEffect(() => {
    if (remaining === 0) onFinished();
  }, [remaining, onFinished]);

  const updateItem = (pos: number, patch: Partial<TestItem>) =>
    setTest((t) => ({ ...t, items: t.items.map((it) => (it.position === pos ? { ...it, ...patch } : it)) }));

  async function choose(option: number) {
    if (busy) return;
    // In exam mode tapping the selected option again clears it (blank).
    const chosen = isExam && item.chosen === option ? null : option;
    const previous = item.chosen;
    const timeMs = Date.now() - shownAt.current;
    shownAt.current = Date.now();
    setError(null);
    updateItem(item.position, { chosen });
    setBusy(!isExam);
    try {
      const sol = await api<Solution | undefined>(`/tests/${test.id}/answer`, {
        method: "POST",
        body: { position: item.position, chosen, time_ms: timeMs },
      });
      if (sol) updateItem(item.position, { solution: sol });
    } catch (err) {
      updateItem(item.position, { chosen: previous });
      if (err instanceof ApiError && err.status === 409) {
        onFinished();
        return;
      }
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  }

  async function finish() {
    const blank = test.items.length - answered;
    const msg =
      isExam && blank > 0
        ? `Quedan ${blank} preguntas en blanco. ¿Entregar el examen?`
        : "¿Terminar el test y ver el resultado?";
    if (!confirm(msg)) return;
    try {
      await api(`/tests/${test.id}/finish`, { method: "POST" });
    } catch (err) {
      if (!(err instanceof ApiError && err.status === 409)) {
        setError(errorMessage(err));
        return;
      }
    }
    setParams({}, { replace: true });
    onFinished();
  }

  async function abandon() {
    if (!confirm("¿Abandonar el test? No se calculará nota, pero las respuestas dadas se guardan.")) return;
    try {
      await api(`/tests/${test.id}/abandon`, { method: "POST" });
      navigate("/tests", { replace: true });
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  return (
    <div className="runner">
      <div className="runner-bar">
        <button type="button" className="link" onClick={() => setShowIndex((v) => !v)}>
          {index + 1}/{test.items.length} ▾
        </button>
        <span className="muted small">
          {modeLabel[test.mode]} · {answered} respondidas
        </span>
        {remaining !== undefined && (
          <span className={remaining < 300 ? "clock low" : "clock"} aria-label="Tiempo restante">
            {formatClock(remaining)}
          </span>
        )}
      </div>
      <div className="progress" aria-hidden>
        <div style={{ width: `${(answered / test.items.length) * 100}%` }} />
      </div>

      {showIndex && <Navigator test={test} current={index} onGo={goTo} />}
      <ErrorBox message={error} />

      <article className="question">
        <p className="stem">{item.stem}</p>
        <Options
          options={item.options}
          chosen={item.chosen}
          correct={item.solution?.correct}
          disabled={busy || practiceDone}
          onChoose={choose}
        />
        {item.solution && <SolutionBox solution={item.solution} chosen={item.chosen} />}
        <FlagControl
          key={item.position}
          testId={test.id}
          item={item}
          onChange={(flagged, note) => updateItem(item.position, { flagged, flag_note: note })}
        />
      </article>

      <div className="actions sticky runner-actions">
        {isExam && (
          <button type="button" disabled={index === 0} onClick={() => goTo(index - 1)}>
            ←
          </button>
        )}
        {isLast || (!isExam && answered === test.items.length) ? (
          <button type="button" className="primary" onClick={finish}>
            {isExam ? "Entregar" : "Ver resultado"}
          </button>
        ) : (
          <button
            type="button"
            className="primary"
            disabled={!isExam && !practiceDone}
            onClick={() => goTo(index + 1)}
          >
            {isExam && item.chosen === null ? "Saltar →" : "Siguiente →"}
          </button>
        )}
      </div>
      <div className="runner-footer">
        {!isLast && (isExam || answered < test.items.length) && (
          <button type="button" className="link" onClick={finish}>
            {isExam ? "Entregar ya" : "Terminar aquí"}
          </button>
        )}
        <button type="button" className="link danger" onClick={abandon}>
          Abandonar
        </button>
      </div>
    </div>
  );
}
