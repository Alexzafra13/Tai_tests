import { useState } from "react";
import { Link, useNavigate } from "react-router";
import { api, errorMessage } from "../../api";
import { formatScore } from "../../format";
import { modeLabel, penaltyLabel, type Test, type TestItem, type TestResult } from "../../types";
import { useAuth } from "../../auth";
import { ErrorBox } from "../../components/Form";
import { Options, SolutionBox } from "../../components/QuestionView";

type ResultFilter = "all" | "wrong" | "blank" | "reported";

const filterLabels: Record<ResultFilter, string> = {
  all: "Todas",
  wrong: "Falladas",
  blank: "En blanco",
  reported: "Dudosas",
};

const matches: Record<ResultFilter, (it: TestItem) => boolean> = {
  all: () => true,
  wrong: (it) => it.solution?.is_correct === false,
  blank: (it) => it.chosen === null,
  reported: (it) => it.reported,
};

export function Results({ test }: { test: Test }) {
  const navigate = useNavigate();
  const { isAdmin } = useAuth();
  const [filter, setFilter] = useState<ResultFilter>(test.result && test.result.wrong > 0 ? "wrong" : "all");
  const [error, setError] = useState<string | null>(null);
  const wrongIds = test.items.filter(matches.wrong).map((it) => it.question_id);

  async function retryWrong() {
    try {
      const { id } = await api<{ id: number }>("/tests", {
        method: "POST",
        body: {
          mode: "practice",
          count: wrongIds.length,
          penalty: test.penalty,
          time_limit_min: 0,
          filters: { topic_ids: [], block_ids: [], source_ids: [], origins: [], question_ids: wrongIds },
        },
      });
      navigate(`/tests/${id}`);
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  return (
    <>
      <div className="page-head">
        <h2>{test.status === "abandoned" ? "Test abandonado" : "Resultado"}</h2>
        <Link to="/tests" className="link">
          Historial
        </Link>
      </div>
      <ErrorBox message={error} />

      {test.result && <ScoreCard result={test.result} mode={test.mode} />}

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
        {(Object.keys(filterLabels) as ResultFilter[]).map((f) => (
          <button key={f} type="button" className={filter === f ? "chip active" : "chip"} onClick={() => setFilter(f)}>
            {filterLabels[f]} ({test.items.filter(matches[f]).length})
          </button>
        ))}
      </div>

      <ul className="list">
        {test.items.filter(matches[filter]).map((it) => (
          <li key={it.position} className="card review-item">
            <p className="stem">
              <span className="muted">{it.position + 1}. </span>
              {it.stem}
            </p>
            <Options options={it.options} chosen={it.chosen} correct={it.solution?.correct} />
            {it.solution && <SolutionBox solution={it.solution} chosen={it.chosen} />}
            <div className="review-links">
              {it.reported && <span className="badge warn">Dudosa</span>}
              {isAdmin && (
                <Link to={`/questions/${it.question_id}`} className="small">
                  Editar pregunta
                </Link>
              )}
            </div>
          </li>
        ))}
      </ul>
    </>
  );
}

function ScoreCard({ result: r, mode }: { result: TestResult; mode: Test["mode"] }) {
  return (
    <section className="card score-card">
      <div className={r.passed ? "score pass" : "score"}>
        {formatScore(r.score)}
        <span className="score-max"> / {formatScore(r.max, 0)}</span>
      </div>
      <p className={r.passed ? "ok" : "error"}>
        <strong>{r.passed ? "Aprobado" : "Por debajo del aprobado"}</strong>
        <span className="muted"> · aprobado en {formatScore(r.pass_mark, 0)}</span>
      </p>
      <div className="score-breakdown">
        <span className="ok">{r.correct} aciertos</span>
        <span className="error">{r.wrong} fallos</span>
        <span className="muted">{r.blank} en blanco</span>
      </div>
      <p className="muted small">
        {modeLabel[mode]} · {penaltyLabel(r.penalty)} · netas {r.net.toLocaleString("es-ES")} de {r.total}
      </p>
      {r.penalty > 0 && r.wrong > 0 && (
        <p className="small">
          Sin penalización habrías sacado <strong>{formatScore(r.score_no_penalty)}</strong>: los fallos te han costado{" "}
          {formatScore(r.score_no_penalty - r.score)} puntos.
        </p>
      )}
    </section>
  );
}
