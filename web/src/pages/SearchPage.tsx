import { useState } from "react";
import { Link, useSearchParams } from "react-router";
import { ErrorBox } from "../components/Form";
import { Options, SourceLink } from "../components/QuestionView";
import { useDebounced, useResource } from "../hooks";
import { originLabel, type SearchHit } from "../types";
import "./SearchPage.css";

// SearchPage finds published questions by any word of the stem, options,
// explanation or reference, ignoring accents. The query lives in the URL so
// going back keeps it.
export function SearchPage() {
  const [params, setParams] = useSearchParams();
  const q = params.get("q") ?? "";
  const query = useDebounced(q.trim(), 250);
  const hits = useResource<SearchHit[]>(query ? `/search?q=${encodeURIComponent(query)}` : null);

  return (
    <>
      <h2>Buscar</h2>
      <input
        type="search"
        autoFocus
        placeholder="Palabras del enunciado, opciones o artículo…"
        aria-label="Buscar preguntas"
        value={q}
        onChange={(e) => setParams(e.target.value ? { q: e.target.value } : {}, { replace: true })}
      />
      <ErrorBox message={hits.error} />
      {query && hits.data && (
        <p className="muted small">
          {hits.data.length === 0
            ? "Sin resultados."
            : hits.data.length >= 30
              ? "Mostrando los 30 mejores resultados; añade palabras para afinar."
              : `${hits.data.length} resultados`}
        </p>
      )}
      <ul className="list search-results">
        {hits.data?.map((h) => (
          <li key={h.id}>
            <SearchResult hit={h} />
          </li>
        ))}
      </ul>
    </>
  );
}

// SearchResult hides the answer until asked, so a search doubles as a quick
// self-test. The stem opens the question on its own page.
function SearchResult({ hit }: { hit: SearchHit }) {
  const [open, setOpen] = useState(false);
  return (
    <article className="card search-hit">
      <Link to={`/search/${hit.id}`} className="stem">
        {hit.stem}
      </Link>
      <Options options={hit.options} correct={open ? hit.correct : undefined} />
      {open ? (
        <div className="solution">
          {hit.explanation && <p>{hit.explanation}</p>}
          <div className="source">
            <span className="muted small">
              {originLabel[hit.origin]} · {hit.source_title} · {hit.source_ref}
            </span>
            {hit.source_quote && <blockquote>{hit.source_quote}</blockquote>}
            <SourceLink kind={hit.source_kind} url={hit.source_url} />
          </div>
        </div>
      ) : (
        <button type="button" className="link" onClick={() => setOpen(true)}>
          Ver respuesta
        </button>
      )}
      <Link to={`/search/${hit.id}`} className="small">
        Abrir pregunta →
      </Link>
    </article>
  );
}
