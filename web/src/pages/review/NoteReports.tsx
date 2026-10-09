import { useState } from "react";
import { Link } from "react-router";
import { api, errorMessage } from "../../api";
import { useResource } from "../../hooks";
import type { NoteReport } from "../../types";
import { ErrorBox } from "../../components/Form";
import { formatDate } from "../../format";

// NoteReports lists what users have said is wrong or missing in the study
// notes. The notes come with the app: a fix is a change to data/notes, so
// here an administrator only reads and closes the reports.
export function NoteReports() {
  const { data, reload } = useResource<NoteReport[]>("/review/notes");
  const [error, setError] = useState<string | null>(null);

  async function resolve(id: number) {
    setError(null);
    try {
      await api(`/review/notes/${id}/resolve`, { method: "POST" });
      reload();
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  if (!data || data.length === 0) return null;
  return (
    <details className="card note-reports" open>
      <summary>
        <strong>Avisos en los apuntes</strong>
        <span className="muted"> · {data.length}</span>
      </summary>
      <ErrorBox message={error} />
      <ul>
        {data.map((r) => (
          <li key={r.id}>
            <p className="muted small">
              <Link to={`/study/${r.topic_id}`} className="link">
                {r.topic_code} · punto {r.point}
              </Link>{" "}
              · {r.username} · {formatDate(r.created_at)}
            </p>
            {r.excerpt && <blockquote>{r.excerpt}</blockquote>}
            <p>{r.note}</p>
            <button type="button" onClick={() => resolve(r.id)}>
              Marcar como resuelto
            </button>
          </li>
        ))}
      </ul>
    </details>
  );
}
