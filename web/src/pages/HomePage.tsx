import { Link } from "react-router";
import { useResource } from "../hooks";
import type { Page, Question } from "../types";

const upcoming = [
  { title: "Test", desc: "Práctica, examen y simulacros oficiales" },
  { title: "Repaso", desc: "Repetición espaciada y preguntas falladas" },
  { title: "Revisión", desc: "Borradores y preguntas marcadas como dudosas" },
  { title: "Estadísticas", desc: "Aciertos por bloque y tema" },
];

export function HomePage() {
  const published = useResource<Page<Question>>("/questions?status=published&limit=1");
  const drafts = useResource<Page<Question>>("/questions?status=draft&limit=1");

  return (
    <>
      <h2>Inicio</h2>
      <ul className="tiles">
        <li>
          <Link to="/questions" className="card tile">
            <strong>Preguntas</strong>
            <span className="muted">
              {published.data?.total ?? "…"} publicadas · {drafts.data?.total ?? "…"} borradores
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
    </>
  );
}
