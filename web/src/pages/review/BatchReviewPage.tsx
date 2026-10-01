import { useEffect, useState } from "react";
import { Link } from "react-router";
import { api, errorMessage } from "../../api";
import { useResource } from "../../hooks";
import { optionLetters, type BatchResult, type Page, type ReviewItem, type Source } from "../../types";
import { ErrorBox, Loading } from "../../components/Form";

// BatchReviewPage accepts many drafts at once, e.g. a whole imported exam.
// Each question is still validated on its own; failures are listed with
// their reason and stay in the queue.
export function BatchReviewPage() {
  const [sourceId, setSourceId] = useState(0);
  const { data: sources } = useResource<Source[]>("/sources");
  const queue = useResource<Page<ReviewItem>>(`/review?kind=drafts&source=${sourceId}&limit=200`);
  const [selected, setSelected] = useState<Set<number>>(new Set());
  const [results, setResults] = useState<BatchResult[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const items = queue.data?.items ?? [];
  const ready = (it: ReviewItem) => it.topic_ids.length > 0 && it.reports.length === 0;

  // Preselect everything that can be accepted as is.
  useEffect(() => {
    setSelected(new Set((queue.data?.items ?? []).filter(ready).map((it) => it.id)));
  }, [queue.data]);

  function toggle(id: number) {
    setSelected((s) => {
      const next = new Set(s);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  async function acceptSelected() {
    if (!confirm(`¿Aceptar ${selected.size} preguntas? Pasarán a salir en los tests.`)) return;
    setBusy(true);
    setError(null);
    try {
      const { results } = await api<{ results: BatchResult[] }>("/review/accept-batch", {
        method: "POST",
        body: { ids: [...selected] },
      });
      setResults(results);
      queue.reload();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  }

  const failed = results?.filter((r) => !r.ok) ?? [];

  return (
    <div>
      <div className="page-head">
        <h2>Aceptar en bloque</h2>
        <Link to="/review" className="link">
          Una a una
        </Link>
      </div>
      <p className="hint">
        Revisa unas cuantas al azar antes de aceptar el resto. Las que no tienen tema o están marcadas como dudosas no se
        preseleccionan.
      </p>

      <select value={sourceId} onChange={(e) => setSourceId(Number(e.target.value))}>
        <option value={0}>Todas las fuentes</option>
        {sources?.map((s) => (
          <option key={s.id} value={s.id}>
            {s.title}
          </option>
        ))}
      </select>

      <ErrorBox message={error ?? queue.error} />
      {results && (
        <div className={failed.length ? "error-box" : "ok-box"}>
          {results.length - failed.length} aceptadas
          {failed.length > 0 && `, ${failed.length} no se pudieron aceptar y siguen en la cola`}.
        </div>
      )}
      {queue.loading && !queue.data && <Loading />}
      {queue.data && items.length === 0 && <p className="muted">No hay borradores.</p>}

      {items.length > 0 && (
        <div className="batch-toolbar">
          <label className="check">
            <input
              type="checkbox"
              checked={selected.size === items.length}
              onChange={(e) => setSelected(new Set(e.target.checked ? items.map((it) => it.id) : []))}
            />
            <span>
              {selected.size} de {items.length} seleccionadas
              {queue.data && queue.data.total > items.length && ` (se muestran las primeras ${items.length})`}
            </span>
          </label>
        </div>
      )}

      <ul className="list">
        {items.map((it) => (
          <li key={it.id} className="card batch-row">
            <label className="check">
              <input type="checkbox" checked={selected.has(it.id)} onChange={() => toggle(it.id)} />
              <span>
                <span className="stem-preview">{it.stem}</span>
                <span className="small">
                  <span className="ok">
                    ✓ {optionLetters[it.correct]}) {it.options[it.correct]}
                  </span>
                  <span className="muted">
                    {" · "}
                    {it.source_ref}
                    {!ready(it) && (it.reports.length > 0 ? " · dudosa" : " · sin tema")}
                  </span>
                </span>
              </span>
            </label>
          </li>
        ))}
      </ul>

      {items.length > 0 && (
        <div className="actions sticky">
          <button type="button" className="primary" disabled={busy || selected.size === 0} onClick={acceptSelected}>
            {busy ? "Aceptando…" : `Aceptar ${selected.size}`}
          </button>
        </div>
      )}
    </div>
  );
}
