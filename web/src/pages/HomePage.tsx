import { useState } from "react";
import { Link, useNavigate } from "react-router";
import { errorMessage } from "../api";
import { useAuth } from "../auth";
import { ErrorBox } from "../components/Form";
import { useResource } from "../hooks";
import type { Page, Question, ReviewCounts, StudySummary, TestFilters, TestSummary } from "../types";
import { QUICK_COUNT, startPractice } from "./test/start";
import { TestRow } from "./TestsPage";

export function HomePage() {
  const { user, isAdmin } = useAuth();
  const inProgress = useResource<TestSummary[]>("/tests?status=in_progress&limit=3");
  const recent = useResource<TestSummary[]>("/tests?status=finished&limit=3");
  const study = useResource<StudySummary>("/study/summary");
  const navigate = useNavigate();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // One tap starts a short practice test on what is pending.
  async function quickTest(filters: Partial<TestFilters>, pending: number) {
    setBusy(true);
    setError(null);
    try {
      navigate(`/tests/${await startPractice(filters, Math.min(pending, QUICK_COUNT))}`);
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  }

  const due = study.data?.review.due ?? 0;
  const failed = study.data?.failed ?? 0;

  return (
    <>
      <h2>Hola, {user?.display_name || user?.username}</h2>

      <TestList title="Continuar" tests={inProgress.data} />
      <ErrorBox message={error} />

      <ul className="tiles">
        <li>
          <Link to="/tests/new" className="card tile primary-tile">
            <strong>Nuevo test</strong>
            <span>Práctica o examen</span>
          </Link>
        </li>
        <li>
          <button className="card tile" disabled={busy || due === 0} onClick={() => quickTest({ due: true }, due)}>
            <strong>Repaso de hoy</strong>
            <span className="muted">{due > 0 ? `${due} por repasar` : study.data ? "Al día" : "Repetición espaciada"}</span>
          </button>
        </li>
        <li>
          <button className="card tile" disabled={busy || failed === 0} onClick={() => quickTest({ failed: true }, failed)}>
            <strong>Falladas</strong>
            <span className="muted">{failed > 0 ? `${failed} por corregir` : "Ninguna pendiente"}</span>
          </button>
        </li>
        <li>
          <Link to="/stats" className="card tile">
            <strong>Estadísticas</strong>
            <span className="muted">Aciertos por tema y progreso</span>
          </Link>
        </li>
        <li>
          <Link to="/search" className="card tile">
            <strong>Buscar</strong>
            <span className="muted">En todas las preguntas</span>
          </Link>
        </li>
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
