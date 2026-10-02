import { ErrorBox } from "../../components/Form";
import { useResource } from "../../hooks";
import type { Stats } from "../../types";
import { ActivityChart } from "./ActivityChart";
import { TopicTable } from "./TopicTable";
import "./stats.css";

export function StatsPage() {
  const { data, error } = useResource<Stats>("/stats");

  return (
    <>
      <h2>Estadísticas</h2>
      <ErrorBox message={error} />
      {data && (
        <>
          <Summary stats={data} />
          <section className="home-section">
            <h3>Últimos 30 días</h3>
            {data.overview.answered > 0 ? (
              <ActivityChart days={data.timeline} />
            ) : (
              <p className="muted">Aún no has respondido preguntas. Haz un test y aquí verás tu progreso.</p>
            )}
          </section>
          <section className="home-section">
            <h3>Por tema</h3>
            {data.topics.length > 0 ? (
              <TopicTable topics={data.topics} />
            ) : (
              <p className="muted">El temario todavía no está cargado.</p>
            )}
          </section>
        </>
      )}
    </>
  );
}

function Summary({ stats }: { stats: Stats }) {
  const o = stats.overview;
  const acc = o.answered > 0 ? `${Math.round((o.correct / o.answered) * 100)} %` : "—";
  const tiles = [
    { value: o.answered, label: "respondidas" },
    { value: acc, label: "aciertos" },
    { value: o.study_days, label: o.study_days === 1 ? "día de estudio" : "días de estudio" },
    { value: stats.review.mastered, label: `dominadas de ${stats.review.tracked}` },
  ];
  return (
    <ul className="stat-tiles">
      {tiles.map((t) => (
        <li key={t.label} className="card">
          <strong>{t.value}</strong>
          <span className="muted small">{t.label}</span>
        </li>
      ))}
    </ul>
  );
}
