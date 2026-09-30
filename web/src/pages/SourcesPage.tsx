import { Link } from "react-router";
import { useResource } from "../hooks";
import { sourceKindLabel, type Source } from "../types";
import { ErrorBox, Loading } from "../components/Form";

export function SourcesPage() {
  const { data: sources, error, loading } = useResource<Source[]>("/sources");

  return (
    <>
      <div className="page-head">
        <h2>Fuentes</h2>
        <Link className="button primary small" to="/sources/new">
          Nueva
        </Link>
      </div>
      <ErrorBox message={error} />
      {loading && <Loading />}
      {sources?.length === 0 && (
        <p className="muted">
          Aún no hay fuentes. Cada pregunta debe citar una: añade primero la ley, el examen oficial o la documentación de
          la que sale.
        </p>
      )}
      <ul className="list">
        {sources?.map((s) => (
          <li key={s.id}>
            <Link to={`/sources/${s.id}`} className="card row">
              <div className="row-main">
                <strong>{s.title}</strong>
                <span className="muted small">
                  {[s.reference, s.version_date && `versión ${s.version_date}`].filter(Boolean).join(" · ")}
                </span>
              </div>
              <div className="row-side">
                <span className="badge">{sourceKindLabel[s.kind]}</span>
                <span className="muted small">
                  {s.questions} preg.{!s.has_text && s.kind !== "inap_exam" && " · sin texto"}
                </span>
              </div>
            </Link>
          </li>
        ))}
      </ul>
    </>
  );
}
