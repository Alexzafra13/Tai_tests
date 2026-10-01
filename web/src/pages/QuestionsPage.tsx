import { Link, useSearchParams } from "react-router";
import { useDebounced, useResource } from "../hooks";
import { originLabel, statusLabel, type Block, type Origin, type Page, type Question, type Status } from "../types";
import { ErrorBox, Loading } from "../components/Form";
import { useState } from "react";

const PAGE_SIZE = 30;

export function QuestionsPage() {
  // Filters live in the URL so they survive navigation to a question and back.
  const [params, setParams] = useSearchParams();
  const [text, setText] = useState(params.get("q") ?? "");
  const debouncedText = useDebounced(text, 300);
  const offset = Number(params.get("offset") ?? 0);

  const query = new URLSearchParams();
  for (const key of ["status", "origin", "topic", "block", "reported"]) {
    const v = params.get(key);
    if (v) query.set(key, v);
  }
  if (debouncedText.trim()) query.set("q", debouncedText.trim());
  query.set("limit", String(PAGE_SIZE));
  query.set("offset", String(offset));

  const { data: page, error, loading } = useResource<Page<Question>>(`/questions?${query}`);
  const { data: blocks } = useResource<Block[]>("/syllabus");
  const topicCode = new Map(blocks?.flatMap((b) => b.topics.map((t) => [t.id, `${b.code}·${t.number}`] as const)));

  function setFilter(key: string, value: string) {
    const next = new URLSearchParams(params);
    if (value) next.set(key, value);
    else next.delete(key);
    next.delete("offset");
    setParams(next, { replace: true });
  }

  function goTo(newOffset: number) {
    const next = new URLSearchParams(params);
    next.set("offset", String(newOffset));
    setParams(next);
    window.scrollTo(0, 0);
  }

  return (
    <>
      <div className="page-head">
        <h2>Preguntas</h2>
        <Link className="button primary small" to="/questions/new">
          Nueva
        </Link>
      </div>

      <div className="filters">
        <input type="search" placeholder="Buscar en enunciado, opciones o artículo…" value={text} onChange={(e) => setText(e.target.value)} />
        <select value={params.get("status") ?? ""} onChange={(e) => setFilter("status", e.target.value)}>
          <option value="">Todos los estados</option>
          {(Object.keys(statusLabel) as Status[]).map((s) => (
            <option key={s} value={s}>
              {statusLabel[s]}
            </option>
          ))}
        </select>
        <select value={params.get("origin") ?? ""} onChange={(e) => setFilter("origin", e.target.value)}>
          <option value="">Todos los orígenes</option>
          {(Object.keys(originLabel) as Origin[]).map((o) => (
            <option key={o} value={o}>
              {originLabel[o]}
            </option>
          ))}
        </select>
        <select value={params.get("topic") ?? ""} onChange={(e) => setFilter("topic", e.target.value)}>
          <option value="">Todos los temas</option>
          {blocks?.map((b) => (
            <optgroup key={b.id} label={b.name}>
              {b.topics.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.number}. {t.title}
                </option>
              ))}
            </optgroup>
          ))}
        </select>
      </div>

      <ErrorBox message={error} />
      {loading && !page && <Loading />}
      {page && <p className="muted small">{page.total} preguntas</p>}

      <ul className="list">
        {page?.items.map((q) => (
          <li key={q.id}>
            <Link to={`/questions/${q.id}`} className="card question-row">
              <p className="stem-preview">{q.stem}</p>
              <div className="meta">
                <span className={`badge status-${q.status}`}>{statusLabel[q.status]}</span>
                <span className="badge">{originLabel[q.origin]}</span>
                {q.annulled && <span className="badge warn">Anulada</span>}
                {q.open_reports > 0 && <span className="badge warn">Dudosa ({q.open_reports})</span>}
                {q.topic_ids.map((id) => (
                  <span key={id} className="badge outline">
                    {topicCode.get(id) ?? "?"}
                  </span>
                ))}
                <span className="muted small">
                  {q.source_title} · {q.source_ref}
                </span>
              </div>
            </Link>
          </li>
        ))}
      </ul>

      {page && page.total > PAGE_SIZE && (
        <div className="pager">
          <button disabled={offset === 0} onClick={() => goTo(Math.max(0, offset - PAGE_SIZE))}>
            ← Anteriores
          </button>
          <span className="muted small">
            {offset + 1}–{Math.min(offset + PAGE_SIZE, page.total)} de {page.total}
          </span>
          <button disabled={offset + PAGE_SIZE >= page.total} onClick={() => goTo(offset + PAGE_SIZE)}>
            Siguientes →
          </button>
        </div>
      )}
    </>
  );
}
