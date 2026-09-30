import { Link } from "react-router";
import { useResource } from "../hooks";
import type { Block } from "../types";
import { ErrorBox, Loading } from "../components/Form";

export function SyllabusPage() {
  const { data: blocks, error, loading } = useResource<Block[]>("/syllabus");

  return (
    <>
      <div className="page-head">
        <h2>Temario</h2>
        <Link to="/sources" className="link">
          Fuentes
        </Link>
      </div>
      <ErrorBox message={error} />
      {loading && <Loading />}
      {blocks?.length === 0 && (
        <div className="card">
          <p>Todavía no hay temario cargado.</p>
          <p className="muted">
            Rellena <code>data/syllabus.json</code> con los bloques y temas de la convocatoria (hay un ejemplo en{" "}
            <code>data/syllabus.example.json</code>) y cárgalo con:
          </p>
          <pre>docker compose exec -T tai tai load-syllabus -file - &lt; data/syllabus.json</pre>
        </div>
      )}
      {blocks?.map((b) => (
        <details key={b.id} className="card block" open>
          <summary>
            <strong>{b.name}</strong>
            <span className="muted"> · {b.topics.length} temas</span>
          </summary>
          <ul className="topic-list">
            {b.topics.map((t) => (
              <li key={t.id}>
                <Link to={`/questions?topic=${t.id}`}>
                  <span className="topic-num">{t.number}.</span>
                  <span className="topic-title">{t.title}</span>
                  <span className="count" title="Publicadas / total">
                    {t.published}/{t.questions}
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        </details>
      ))}
    </>
  );
}
