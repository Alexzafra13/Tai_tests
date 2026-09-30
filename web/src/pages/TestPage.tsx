import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router";
import { api, ApiError, errorMessage } from "../api";
import { useResource } from "../hooks";
import { formatClock, formatScore } from "../format";
import { modeLabel, penaltyLabel, type Solution, type Test, type TestItem } from "../types";
import { ErrorBox, Loading } from "../components/Form";
import { FlagControl, Options, SolutionBox } from "../components/QuestionView";

export function TestPage() {
  const { id } = useParams();
  const { data, error, loading, reload } = useResource<Test>(`/tests/${id}`);

  if (loading && !data) return <Loading />;
  if (error || !data) return <ErrorBox message={error ?? "Test no encontrado"} />;
  // Keyed by id so moving to another test (e.g. "retry wrong") resets state.
  if (data.status === "in_progress") return <Runner key={data.id} initial={data} onFinished={reload} />;
  return <Results key={data.id} test={data} />;
}

// --- Runner ------------------------------------------------------------------

function Runner({ initial, onFinished }: { initial: Test; onFinished: () => void }) {
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

  const isLast = index === test.items.length - 1;
  const practiceDone = !isExam && item.solution !== undefined;

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
        <Options item={item} disabled={busy || practiceDone} onChoose={choose} />
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
        {isExam && !isLast && (
          <button type="button" className="link" onClick={finish}>
            Entregar ya
          </button>
        )}
        {!isExam && !isLast && answered < test.items.length && (
          <button type="button" className="link" onClick={finish}>
            Terminar aquí
          </button>
        )}
        <button type="button" className="link danger" onClick={abandon}>
          Abandonar
        </button>
      </div>
    </div>
  );
}

// useCountdown ticks down from the server's remaining seconds. It anchors
// on a local deadline so a sleeping phone shows the right time on wake.
function useCountdown(initialSec: number | undefined): number | undefined {
  const [deadline] = useState(() => (initialSec === undefined ? undefined : Date.now() + initialSec * 1000));
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    if (deadline === undefined) return;
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [deadline]);
  if (deadline === undefined) return undefined;
  return Math.max(0, Math.round((deadline - now) / 1000));
}

function Navigator({ test, current, onGo }: { test: Test; current: number; onGo: (i: number) => void }) {
  return (
    <div className="navigator card">
      {test.items.map((it, i) => {
        let cls = "nav-cell";
        if (it.solution) cls += it.solution.is_correct ? " is-correct" : " is-wrong";
        else if (it.chosen !== null) cls += " is-answered";
        if (it.flagged) cls += " is-flagged";
        if (i === current) cls += " is-current";
        return (
          <button key={i} type="button" className={cls} onClick={() => onGo(i)}>
            {i + 1}
          </button>
        );
      })}
      <p className="muted small legend">
        <span className="nav-cell is-answered" /> respondida <span className="nav-cell is-flagged" /> dudosa
      </p>
    </div>
  );
}

// --- Results -----------------------------------------------------------------

type ResultFilter = "all" | "wrong" | "blank" | "flagged";

function Results({ test }: { test: Test }) {
  const navigate = useNavigate();
  const [filter, setFilter] = useState<ResultFilter>(test.result && test.result.wrong > 0 ? "wrong" : "all");
  const [error, setError] = useState<string | null>(null);
  const r = test.result;

  const wrongIds = test.items.filter((it) => it.solution?.is_correct === false).map((it) => it.question_id);
  const shown = test.items.filter((it) => {
    switch (filter) {
      case "wrong":
        return it.solution?.is_correct === false;
      case "blank":
        return it.chosen === null;
      case "flagged":
        return it.flagged;
      default:
        return true;
    }
  });

  async function retryWrong() {
    try {
      const { id } = await api<{ id: number }>("/tests", {
        method: "POST",
        body: {
          mode: "practice",
          count: wrongIds.length,
          penalty: 0,
          time_limit_min: 0,
          filters: { topic_ids: [], block_ids: [], source_ids: [], origins: [], question_ids: wrongIds },
        },
      });
      navigate(`/tests/${id}`);
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  const counts: Record<ResultFilter, number> = {
    all: test.items.length,
    wrong: wrongIds.length,
    blank: test.items.filter((it) => it.chosen === null).length,
    flagged: test.items.filter((it) => it.flagged).length,
  };
  const labels: Record<ResultFilter, string> = { all: "Todas", wrong: "Falladas", blank: "En blanco", flagged: "Dudosas" };

  return (
    <>
      <div className="page-head">
        <h2>{test.status === "abandoned" ? "Test abandonado" : "Resultado"}</h2>
        <Link to="/tests" className="link">
          Historial
        </Link>
      </div>
      <ErrorBox message={error} />

      {r && (
        <section className="card score-card">
          <div className={r.score >= 5 ? "score pass" : "score"}>{formatScore(r.score)}</div>
          <div className="score-breakdown">
            <span className="ok">{r.correct} aciertos</span>
            <span className="error">{r.wrong} fallos</span>
            <span className="muted">{r.blank} en blanco</span>
          </div>
          <p className="muted small">
            {modeLabel[test.mode]} · {penaltyLabel(r.penalty)} · netas {r.net.toLocaleString("es-ES")} de {r.total}
          </p>
          {r.penalty > 0 && r.wrong > 0 && (
            <p className="small">
              Sin penalización habrías sacado <strong>{formatScore(r.score_no_penalty)}</strong>: los fallos te han
              costado {formatScore(r.score_no_penalty - r.score)} puntos.
            </p>
          )}
        </section>
      )}

      <div className="actions">
        <Link className="button primary" to="/tests/new">
          Nuevo test
        </Link>
        {wrongIds.length > 0 && (
          <button type="button" onClick={retryWrong}>
            Repetir falladas ({wrongIds.length})
          </button>
        )}
      </div>

      <div className="chips result-filters">
        {(Object.keys(labels) as ResultFilter[]).map((f) => (
          <button key={f} type="button" className={filter === f ? "chip active" : "chip"} onClick={() => setFilter(f)}>
            {labels[f]} ({counts[f]})
          </button>
        ))}
      </div>

      <ul className="list">
        {shown.map((it) => (
          <li key={it.position} className="card review-item">
            <p className="stem">
              <span className="muted">{it.position + 1}. </span>
              {it.stem}
            </p>
            <Options item={it} />
            {it.solution && <SolutionBox solution={it.solution} chosen={it.chosen} />}
            <div className="review-links">
              {it.flagged && <span className="badge warn">Dudosa</span>}
              <Link to={`/questions/${it.question_id}`} className="small">
                Editar pregunta
              </Link>
            </div>
          </li>
        ))}
      </ul>
    </>
  );
}
