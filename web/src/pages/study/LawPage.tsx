import { useEffect, useMemo, useState } from "react";
import { Link, useLocation, useNavigate, useParams } from "react-router";
import { useResource } from "../../hooks";
import type { LawText, LawTextSection, QuestionBrief } from "../../types";
import { ErrorBox, Loading } from "../../components/Form";
import { Options } from "../../components/QuestionView";
import { formatDay, lawOrigin, lawShortName } from "../../format";
import "./study.css";

// fold makes searching accent- and case-insensitive ("articulo" finds "Artículo").
function fold(s: string): string {
  return s
    .normalize("NFD")
    .replace(/\p{M}/gu, "")
    .toLowerCase();
}

// LawPage shows a law as published by the BOE, limited to the part of the
// topic when there is one, with the official questions next to the articles
// they cite. A #block in the URL (from a question) opens at that article.
export function LawPage() {
  const { topicId, lawId } = useParams();
  const { hash, key } = useLocation();
  const navigate = useNavigate();
  const { data: law, error, loading } = useResource<LawText>(
    topicId ? `/laws/${lawId}?topic=${topicId}` : `/laws/${lawId}`,
  );
  const [query, setQuery] = useState("");
  const [onlyAsked, setOnlyAsked] = useState(false);
  const target = decodeURIComponent(hash.slice(1));
  // key is "default" when the page was opened directly, with nothing to go back to.
  const fromQuestion = target !== "" && key !== "default";

  useEffect(() => {
    if (law && target) document.getElementById(target)?.scrollIntoView();
  }, [law, target]);

  const visible = useMemo(() => {
    if (!law) return [];
    const q = fold(query.trim());
    if (!q && !onlyAsked) return law.sections;
    return law.sections.filter(
      (s) =>
        s.kind !== "heading" &&
        (!onlyAsked || s.questions.length > 0) &&
        (!q || fold(`${s.title}\n${s.body}\n${s.upcoming?.body ?? ""}`).includes(q)),
    );
  }, [law, query, onlyAsked]);

  const headings = law?.sections.filter((s) => s.kind === "heading" || s.kind === "text") ?? [];
  const asked = law?.sections.filter((s) => s.questions.length > 0).length ?? 0;
  const filtering = query.trim() !== "" || onlyAsked;

  return (
    <>
      <div className="page-head">
        <h2>Estudiar</h2>
        {fromQuestion ? (
          <button type="button" className="link" onClick={() => navigate(-1)}>
            Volver a la pregunta
          </button>
        ) : (
          <Link to={topicId ? `/study/${topicId}` : "/syllabus"} className="link">
            {topicId ? "Volver al tema" : "Temario"}
          </Link>
        )}
      </div>
      <ErrorBox message={error} />
      {loading && <Loading />}
      {law && (
        <>
          <h3 className="law-heading">{lawShortName(law.title) || law.title}</h3>
          {lawShortName(law.title) && <p className="muted law-source">{law.title}</p>}
          <p className="muted law-source">
            {lawOrigin(law.reference).text} ({law.reference}), versión del {formatDay(law.version_date)}.{" "}
            <a href={law.url} target="_blank" rel="noreferrer" className="link">
              Ver en {lawOrigin(law.reference).site}
            </a>
          </p>

          {law.questions.length > 0 && (
            <details className="card asked-general">
              <summary>
                <strong>Preguntas de examen sobre esta norma</strong>
                <span className="muted"> · {law.questions.length}</span>
              </summary>
              {law.questions.map((q) => (
                <AskedQuestion key={q.id} q={q} />
              ))}
            </details>
          )}

          <div className="law-tools">
            <input
              type="search"
              placeholder="Buscar en el texto"
              aria-label="Buscar en el texto de la norma"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
            {asked > 0 && (
              <label className="check">
                <input type="checkbox" checked={onlyAsked} onChange={(e) => setOnlyAsked(e.target.checked)} />
                Solo artículos preguntados ({asked})
              </label>
            )}
          </div>

          {!filtering && headings.length > 1 && (
            <details className="card law-index">
              <summary>
                <strong>Índice</strong>
              </summary>
              <ul>
                {headings.map((h) => (
                  <li key={h.id} className={`lvl-${h.level ?? 1}`}>
                    <a
                      href={`#${h.id}`}
                      onClick={(e) => {
                        e.preventDefault();
                        document.getElementById(h.id)?.scrollIntoView({ behavior: "smooth" });
                      }}
                    >
                      {h.title}
                    </a>
                  </li>
                ))}
              </ul>
            </details>
          )}

          {filtering && <p className="muted">{visible.length === 1 ? "1 resultado" : `${visible.length} resultados`}</p>}
          <div className="law-text">
            {visible.map((s) => (
              <Section key={s.id} s={s} target={s.id === target} />
            ))}
          </div>
        </>
      )}
    </>
  );
}

function Paragraphs({ text }: { text: string }) {
  return (
    <>
      {text.split("\n\n").map((p, i) => (
        <p key={i}>{p}</p>
      ))}
    </>
  );
}

function Section({ s, target }: { s: LawTextSection; target: boolean }) {
  if (s.kind === "heading") {
    const H = (s.level ?? 1) <= 1 ? "h3" : "h4";
    return (
      <H id={s.id} className={`law-h lvl-${s.level ?? 1}`}>
        {s.title}
      </H>
    );
  }
  if (s.kind === "text") {
    return (
      <details id={s.id} className="law-preamble">
        <summary>{s.title}</summary>
        <Paragraphs text={s.body} />
      </details>
    );
  }
  return (
    <article
      id={s.id}
      className={["law-article", s.questions.length > 0 && "asked", target && "target"].filter(Boolean).join(" ")}
    >
      <h5>{s.title}</h5>
      {s.body ? <Paragraphs text={s.body} /> : <p className="muted">No está en vigor todavía.</p>}
      {s.upcoming && (
        <details className="law-upcoming">
          <summary>Nueva redacción en vigor desde el {formatDay(s.upcoming.date)}</summary>
          <Paragraphs text={s.upcoming.body} />
        </details>
      )}
      {s.notes && (
        <details className="law-notes">
          <summary>Modificaciones</summary>
          <p>{s.notes}</p>
        </details>
      )}
      {s.questions.length > 0 && (
        <details className="law-asked" open={target || undefined}>
          <summary>
            Preguntado en {s.questions.length === 1 ? "1 examen" : `${s.questions.length} exámenes`}
          </summary>
          {s.questions.map((q) => (
            <AskedQuestion key={q.id} q={q} />
          ))}
        </details>
      )}
    </article>
  );
}

// AskedQuestion lets you answer an official question before seeing the key.
function AskedQuestion({ q }: { q: QuestionBrief }) {
  const [chosen, setChosen] = useState<number | null>(null);
  return (
    <div className="asked-question">
      <p className="muted asked-ref">{q.source_ref}</p>
      <p className="asked-stem">{q.stem}</p>
      <Options
        options={q.options}
        chosen={chosen}
        correct={chosen === null ? undefined : q.correct}
        onChoose={chosen === null ? setChosen : undefined}
      />
    </div>
  );
}
