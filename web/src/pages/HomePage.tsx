import { Link } from "react-router";
import { useResource } from "../hooks";
import type { Page, Question, ReviewCounts, TestSummary } from "../types";
import { TestRow } from "./TestsPage";

const upcoming = [
  { title: "Repaso", desc: "Repetición espaciada y preguntas falladas" },
  { title: "Estadísticas", desc: "Aciertos por bloque y tema" },
];

export function HomePage() {
  const inProgress = useResource<TestSummary[]>("/tests?status=in_progress&limit=3");
  const recent = useResource<TestSummary[]>("/tests?status=finished&limit=3");
  const published = useResource<Page<Question>>("/questions?status=published&limit=1");
  const review = useResource<ReviewCounts>("/review/counts");

  return (
    <>
      <h2>Inicio</h2>

      {inProgress.data && inProgress.data.length > 0 && (
        <section className="home-section">
          <h3>Continuar</h3>
          <ul className="list">
            {inProgress.data.map((t) => (
              <li key={t.id}>
                <TestRow test={t} />
              </li>
            ))}
          </ul>
        </section>
      )}

      <ul className="tiles">
        <li>
          <Link to="/tests/new" className="card tile primary-tile">
            <strong>Nuevo test</strong>
            <span>{published.data ? `${published.data.total} preguntas publicadas` : "Práctica o examen"}</span>
          </Link>
        </li>
        <li>
          <Link to="/review" className="card tile">
            <strong>Revisión</strong>
            <span className="muted">
              {review.data
                ? review.data.total === 0
                  ? "Nada pendiente"
                  : `${review.data.total} pendientes · ${review.data.flagged} dudosas`
                : "Borradores y dudosas"}
            </span>
          </Link>
        </li>
        <li>
          <Link to="/questions/new" className="card tile">
            <strong>Nueva pregunta</strong>
            <span className="muted">Alta manual con fuente y cita verificada</span>
          </Link>
        </li>
        {upcoming.map((s) => (
          <li key={s.title} className="card tile disabled">
            <strong>{s.title}</strong>
            <span className="muted">{s.desc}</span>
            <span className="badge">Próximamente</span>
          </li>
        ))}
      </ul>

      {recent.data && recent.data.length > 0 && (
        <section className="home-section">
          <h3>Últimos resultados</h3>
          <ul className="list">
            {recent.data.map((t) => (
              <li key={t.id}>
                <TestRow test={t} />
              </li>
            ))}
          </ul>
        </section>
      )}
    </>
  );
}
