import { Link } from "react-router";
import { useAuth } from "../auth";
import { useResource } from "../hooks";
import type { Page, Question, ReviewCounts, TestSummary } from "../types";
import { TestRow } from "./TestsPage";

const upcoming = [
  { title: "Repaso", desc: "Repetición espaciada y preguntas falladas" },
  { title: "Estadísticas", desc: "Aciertos por bloque y tema" },
];

export function HomePage() {
  const { user, isAdmin } = useAuth();
  const inProgress = useResource<TestSummary[]>("/tests?status=in_progress&limit=3");
  const recent = useResource<TestSummary[]>("/tests?status=finished&limit=3");

  return (
    <>
      <h2>Hola, {user?.display_name || user?.username}</h2>

      <TestList title="Continuar" tests={inProgress.data} />

      <ul className="tiles">
        <li>
          <Link to="/tests/new" className="card tile primary-tile">
            <strong>Nuevo test</strong>
            <span>Práctica o examen</span>
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

      {isAdmin && <AdminTiles />}

      <TestList title="Últimos resultados" tests={recent.data} />
    </>
  );
}

function TestList({ title, tests }: { title: string; tests: TestSummary[] | undefined }) {
  if (!tests || tests.length === 0) return null;
  return (
    <section className="home-section">
      <h3>{title}</h3>
      <ul className="list">
        {tests.map((t) => (
          <li key={t.id}>
            <TestRow test={t} />
          </li>
        ))}
      </ul>
    </section>
  );
}

// AdminTiles is only rendered for administrators, so these admin-only
// endpoints are never requested by other users.
function AdminTiles() {
  const review = useResource<ReviewCounts>("/review/counts");
  const published = useResource<Page<Question>>("/questions?status=published&limit=1");

  let reviewText = "Borradores y dudas";
  if (review.data) {
    reviewText =
      review.data.total === 0 ? "Nada pendiente" : `${review.data.total} pendientes · ${review.data.reported} dudosas`;
  }

  return (
    <section className="home-section">
      <h3>Administración</h3>
      <ul className="tiles">
        <li>
          <Link to="/review" className="card tile">
            <strong>Revisión</strong>
            <span className="muted">{reviewText}</span>
          </Link>
        </li>
        <li>
          <Link to="/questions" className="card tile">
            <strong>Preguntas</strong>
            <span className="muted">{published.data ? `${published.data.total} publicadas` : "Banco de preguntas"}</span>
          </Link>
        </li>
        <li>
          <Link to="/users" className="card tile">
            <strong>Usuarios</strong>
            <span className="muted">Cuentas y permisos</span>
          </Link>
        </li>
      </ul>
    </section>
  );
}
