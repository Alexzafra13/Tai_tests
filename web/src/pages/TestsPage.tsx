import { Link } from "react-router";
import { useResource } from "../hooks";
import { formatDate, formatScore } from "../format";
import { modeLabel, type TestSummary } from "../types";
import { ErrorBox, Loading } from "../components/Form";

export function TestsPage() {
  const { data: tests, error, loading } = useResource<TestSummary[]>("/tests?limit=50");

  return (
    <>
      <div className="page-head">
        <h2>Tests</h2>
        <Link className="button primary small" to="/tests/new">
          Nuevo
        </Link>
      </div>
      <ErrorBox message={error} />
      {loading && <Loading />}
      {tests?.length === 0 && <p className="muted">Todavía no has hecho ningún test.</p>}
      <ul className="list">
        {tests?.map((t) => (
          <li key={t.id}>
            <TestRow test={t} />
          </li>
        ))}
      </ul>
    </>
  );
}

export function TestRow({ test: t }: { test: TestSummary }) {
  return (
    <Link to={`/tests/${t.id}`} className="card row">
      <div className="row-main">
        <strong>
          {modeLabel[t.mode]} · {t.total} preguntas
        </strong>
        <span className="muted small">
          {formatDate(t.started_at)}
          {t.status === "in_progress" && ` · ${t.answered} respondidas`}
          {t.status === "finished" && ` · ${t.correct} aciertos, ${t.wrong} fallos`}
        </span>
      </div>
      <div className="row-side">
        {t.status === "in_progress" && <span className="badge warn">En curso</span>}
        {t.status === "abandoned" && <span className="badge">Abandonado</span>}
        {t.status === "finished" && t.score !== null && (
          <strong className={t.passed ? "ok" : "error"}>{formatScore(t.score)}</strong>
        )}
      </div>
    </Link>
  );
}
