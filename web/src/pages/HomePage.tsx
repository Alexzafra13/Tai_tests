import { useState } from "react";
import { Link, useNavigate } from "react-router";
import { errorMessage } from "../api";
import { useAuth } from "../auth";
import { ErrorBox } from "../components/Form";
import { useResource } from "../hooks";
import {
  modeLabel,
  type Page,
  type Question,
  type ReviewCounts,
  type Stats,
  type TestFilters,
  type TestSummary,
  type TopicStats,
} from "../types";
import { QUICK_COUNT, startPractice } from "./test/start";
import { TestRow } from "./TestsPage";
import "./home.css";

const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;
const today = new Intl.DateTimeFormat("es-ES", { weekday: "long", day: "numeric", month: "long" });

function greeting(hour: number) {
  if (hour >= 6 && hour < 14) return "Buenos días";
  if (hour >= 14 && hour < 21) return "Buenas tardes";
  return "Buenas noches";
}

// A review question takes about 20 seconds.
const minutesFor = (questions: number) => Math.max(1, Math.round(questions / 3));

export function HomePage() {
  const { user, isAdmin } = useAuth();
  const inProgress = useResource<TestSummary[]>("/tests?status=in_progress&limit=1");
  const recent = useResource<TestSummary[]>("/tests?status=finished&limit=3");
  const stats = useResource<Stats>(`/stats?tz=${encodeURIComponent(zone)}`);
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

  const now = new Date();
  const due = stats.data?.review.due ?? 0;
  const failed = stats.data?.failed ?? 0;
  const current = inProgress.data?.[0];
  const dateText = today.format(now);

  return (
    <>
      <p className="home-date">{dateText.charAt(0).toUpperCase() + dateText.slice(1)}</p>
      <h1 className="home-hello">
        {greeting(now.getHours())}, {user?.display_name || user?.username}
      </h1>
      <ErrorBox message={error ?? stats.error} />

      <section className="card hero">
        <span className="eyebrow">Repaso de hoy</span>
        {due > 0 ? (
          <>
            <div className="hero-row">
              <span className="hero-num">{due}</span>
              <span className="hero-unit">
                {due === 1 ? "pregunta" : "preguntas"}
                <br />
                <span className="muted">unos {minutesFor(Math.min(due, QUICK_COUNT))} min</span>
              </span>
            </div>
            <button className="primary hero-action" disabled={busy} onClick={() => quickTest({ due: true }, due)}>
              Empezar repaso
            </button>
          </>
        ) : (
          <>
            <p className="hero-empty">
              {stats.data?.review.tracked ? "Estás al día. Mañana habrá más." : "Aún no hay nada que repasar."}
            </p>
            <p className="muted small">
              Cada pregunta que respondes vuelve justo antes de que se te olvide.
            </p>
            <Link to="/tests/new" className="button primary hero-action">
              Hacer un test
            </Link>
          </>
        )}
      </section>

      <div className="pair">
        <button
          type="button"
          className="card mini"
          disabled={busy || failed === 0}
          onClick={() => quickTest({ failed: true }, failed)}
        >
          <span className="eyebrow">Falladas</span>
          <strong>{failed}</strong>
          <span className="muted small">{failed > 0 ? "por corregir" : "ninguna pendiente"}</span>
        </button>
        <Link to="/tests/new" className="card mini">
          <span className="eyebrow">Nuevo test</span>
          <strong aria-hidden>+</strong>
          <span className="muted small">práctica o examen</span>
        </Link>
      </div>

      {current && (
        <Link to={`/tests/${current.id}`} className="card resume">
          <span>
            <span className="eyebrow">Continuar</span>
            <span className="resume-title">
              {modeLabel[current.mode]} · {current.total} preguntas
            </span>
          </span>
          <span className="resume-count">
            {current.answered}
            <small>/{current.total}</small>
          </span>
        </Link>
      )}

      {stats.data && <BlockProgress topics={stats.data.topics} />}

      {isAdmin && <AdminTiles />}

      <TestList title="Últimos resultados" tests={recent.data} />
    </>
  );
}

const roman = ["I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X"];

// BlockProgress shows accuracy per syllabus block, a summary of the
// statistics page.
function BlockProgress({ topics }: { topics: TopicStats[] }) {
  const blocks: { id: number; name: string; answered: number; correct: number }[] = [];
  for (const t of topics) {
    let b = blocks.find((x) => x.id === t.block_id);
    if (!b) blocks.push((b = { id: t.block_id, name: t.block_name, answered: 0, correct: 0 }));
    b.answered += t.answered;
    b.correct += t.correct;
  }
  if (blocks.length === 0) return null;

  return (
    <section className="home-section">
      <div className="section-head">
        <h3>Por bloque</h3>
        <Link to="/stats" className="small">
          Ver progreso
        </Link>
      </div>
      <ul className="blocks">
        {blocks.map((b, i) => {
          const pct = b.answered > 0 ? Math.round((b.correct / b.answered) * 100) : null;
          return (
            <li key={b.id} className="blk">
              <span className="blk-n">{roman[i] ?? i + 1}</span>
              <span className="blk-t">{b.name}</span>
              <span className="blk-p">{pct === null ? "—" : `${pct} %`}</span>
              <span className="bar" role="img" aria-label={pct === null ? "Sin responder" : `${pct} % de aciertos`}>
                {pct !== null && <i style={{ width: `${pct}%` }} />}
              </span>
            </li>
          );
        })}
      </ul>
    </section>
  );
}

function TestList({ title, tests }: { title: string; tests: TestSummary[] | undefined }) {
  if (!tests || tests.length === 0) return null;
  return (
    <section className="home-section">
      <div className="section-head">
        <h3>{title}</h3>
        <Link to="/tests" className="small">
          Historial
        </Link>
      </div>
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
            <span className="muted small">{reviewText}</span>
          </Link>
        </li>
        <li>
          <Link to="/questions" className="card tile">
            <strong>Preguntas</strong>
            <span className="muted small">
              {published.data ? `${published.data.total} publicadas` : "Banco de preguntas"}
            </span>
          </Link>
        </li>
        <li>
          <Link to="/stats" className="card tile">
            <strong>Progreso</strong>
            <span className="muted small">Tus estadísticas</span>
          </Link>
        </li>
        <li>
          <Link to="/search" className="card tile">
            <strong>Buscar</strong>
            <span className="muted small">En todas las preguntas</span>
          </Link>
        </li>
        <li>
          <Link to="/users" className="card tile">
            <strong>Usuarios</strong>
            <span className="muted small">Cuentas y permisos</span>
          </Link>
        </li>
      </ul>
    </section>
  );
}
