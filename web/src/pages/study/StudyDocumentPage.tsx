import { useEffect, useRef, useState } from "react";
import { Link, useParams } from "react-router";
import { useResource } from "../../hooks";
import type { NotePoint, NoteRef, StudyTopic, TopicNote } from "../../types";
import { ErrorBox, Loading } from "../../components/Form";
import { loadPref, savePref } from "../../storage";
import tokensCss from "../../styles/tokens.css?raw";
import documentCss from "./document.css?raw";
import { Bold } from "./NoteView";
import "./document.css";

const QUOTES_KEY = "tai.doc.quotes";

// A law label is "Artículo 62 · Constitución Española"; a page label is its
// title. Sources groups a point's refs by law: "Constitución Española, arts.
// 62 y 63".
function sources(refs: NoteRef[]): string[] {
  const out: { name: string; parts: string[] }[] = [];
  for (const r of refs) {
    if (r.url) {
      out.push({ name: r.label, parts: [] });
      continue;
    }
    const [head, ...rest] = r.label.split(" · ");
    const name = rest.join(" · ") || head;
    const part = rest.length ? head.replace(/^Artículo /, "") : "";
    const g = out.find((o) => o.name === name);
    if (!g) out.push({ name, parts: part ? [part] : [] });
    else if (part && !g.parts.includes(part)) g.parts.push(part);
  }
  return out.map(({ name, parts }) => {
    if (parts.length === 0) return name;
    const arts = parts.every((p) => /^\d/.test(p));
    const list = parts.length > 1 ? `${parts.slice(0, -1).join(", ")} y ${parts[parts.length - 1]}` : parts[0];
    return `${name}, ${arts ? (parts.length > 1 ? "arts. " : "art. ") : ""}${list}`;
  });
}

function asked(refs: NoteRef[]): number {
  return refs.reduce((n, r) => Math.max(n, r.asked), 0);
}

function Point({ p, quotes }: { p: NotePoint; quotes: boolean }) {
  const n = asked(p.refs);
  return (
    <li className={n > 0 ? "doc-point asked" : "doc-point"}>
      <p className="doc-text">
        <Bold text={p.text} />
      </p>
      {p.items.length > 0 && (
        <ul className="doc-items">
          {p.items.map((it) => (
            <Point key={it.path} p={it} quotes={quotes} />
          ))}
        </ul>
      )}
      {p.refs.length > 0 && (
        <p className="doc-src">
          {sources(p.refs).join(" · ")}
          {n > 0 && <span className="doc-asked">★ {n === 1 ? "1 pregunta de examen" : `${n} preguntas de examen`}</span>}
        </p>
      )}
      {quotes && p.refs.length > 0 && (
        <div className="doc-quotes">
          {p.refs.map((r, i) => (
            <figure key={i}>
              {r.quotes.map((q, j) => (
                <blockquote key={j}>«{q}»</blockquote>
              ))}
              <figcaption>
                {r.url ? (
                  <a href={r.url} target="_blank" rel="noreferrer">
                    {r.label}
                  </a>
                ) : (
                  r.label
                )}
              </figcaption>
            </figure>
          ))}
        </div>
      )}
    </li>
  );
}

function allRefs(note: TopicNote): NoteRef[] {
  const out: NoteRef[] = [];
  const walk = (p: NotePoint) => {
    out.push(...p.refs);
    p.items.forEach(walk);
  };
  note.sections.forEach((s) => s.points.forEach(walk));
  return out;
}

function countPoints(points: NotePoint[]): number {
  return points.reduce((n, p) => n + 1 + countPoints(p.items), 0);
}

// The laws and pages the note rests on, laws first, each once.
function bibliography(note: TopicNote) {
  const laws = new Set<string>();
  const pages = new Map<string, string>();
  for (const r of allRefs(note)) {
    if (r.url) pages.set(r.url, r.label);
    else laws.add(r.label.split(" · ").slice(1).join(" · ") || r.label);
  }
  return { laws: [...laws], pages: [...pages].map(([url, title]) => ({ url, title })) };
}

function Document({ topic, note, quotes }: { topic: StudyTopic; note: TopicNote; quotes: boolean }) {
  const bib = bibliography(note);
  const points = note.sections.reduce((n, s) => n + countPoints(s.points), 0);
  return (
    <>
      <header className="doc-cover">
        <p className="doc-kicker">{topic.block}</p>
        <p className="doc-number">Tema {topic.number}</p>
        <h1 className="doc-title">{topic.title}</h1>
        <p className="doc-meta">
          Apuntes · {note.sections.length} apartados · {points} puntos · {bib.laws.length + bib.pages.length} fuentes
        </p>
        <p className="doc-legend">
          Bajo cada punto, la norma o la página oficial de donde sale. <span className="doc-asked">★</span> marca los
          artículos que ya han caído en exámenes del INAP.
        </p>
      </header>
      <nav className="doc-toc" aria-label="Índice">
        <h2>Índice</h2>
        <ol>
          {note.sections.map((s, i) => (
            <li key={i}>
              <a href={`#apartado-${i + 1}`}>{s.title}</a>
            </li>
          ))}
        </ol>
      </nav>
      {note.sections.map((s, i) => (
        <section key={i} className="doc-section" id={`apartado-${i + 1}`}>
          <h2>
            <span className="doc-secnum">{i + 1}</span>
            {s.title}
          </h2>
          <ul className="doc-points">
            {s.points.map((p) => (
              <Point key={p.path} p={p} quotes={quotes} />
            ))}
          </ul>
        </section>
      ))}
      <footer className="doc-bib">
        <h2>Fuentes</h2>
        {bib.laws.length > 0 && (
          <>
            <h3>Normas (texto consolidado del BOE)</h3>
            <ul>
              {bib.laws.map((l) => (
                <li key={l}>{l}</li>
              ))}
            </ul>
          </>
        )}
        {bib.pages.length > 0 && (
          <>
            <h3>Páginas oficiales</h3>
            <ul>
              {bib.pages.map((p) => (
                <li key={p.url}>
                  {p.title}
                  <br />
                  <a href={p.url}>{p.url}</a>
                </li>
              ))}
            </ul>
          </>
        )}
        <p className="doc-colophon">TAI Go · apuntes del tema {topic.code}</p>
      </footer>
    </>
  );
}

// fontFaces inlines the Latin faces of the app's fonts, so the downloaded
// file looks the same offline. Without them it falls back to system fonts.
// Browsers write the Latin range as "U+0000-00FF" or "U+0-FF".
const LATIN = /U\+0+-0*FF(?![0-9A-F])/i;

async function fontFaces(): Promise<string> {
  const rules: CSSFontFaceRule[] = [];
  for (const sheet of Array.from(document.styleSheets)) {
    try {
      for (const r of Array.from(sheet.cssRules)) {
        if (r instanceof CSSFontFaceRule && LATIN.test(r.style.getPropertyValue("unicode-range"))) rules.push(r);
      }
    } catch {
      // A sheet from another origin can't be read.
    }
  }
  const faces = await Promise.all(
    rules.map(async (r) => {
      const url = /url\(["']?([^"')]+)/.exec(r.style.getPropertyValue("src"))?.[1];
      if (!url) return "";
      try {
        const blob = await (await fetch(new URL(url, r.parentStyleSheet?.href || location.href))).blob();
        const data = await new Promise<string>((done, fail) => {
          const fr = new FileReader();
          fr.onload = () => done(String(fr.result));
          fr.onerror = () => fail(fr.error);
          fr.readAsDataURL(blob);
        });
        return r.cssText.replace(/src:[^;]+;/, `src: url(${data}) format("woff2");`);
      } catch {
        return "";
      }
    }),
  );
  return faces.join("\n");
}

// download saves the document as a single HTML file that opens offline in
// any browser, light theme, with the same styles as the page.
async function download(topic: StudyTopic, article: HTMLElement) {
  const title = `Tema ${topic.number}. ${topic.title}`;
  const esc = (s: string) => s.replace(/&/g, "&amp;").replace(/</g, "&lt;");
  const html = `<!doctype html>
<html lang="es" data-theme="light">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${esc(title)}</title>
<style>${await fontFaces()}
${tokensCss}
${documentCss}</style>
</head>
<body class="doc-page">
<article class="doc">${article.innerHTML}</article>
</body>
</html>
`;
  const url = URL.createObjectURL(new Blob([html], { type: "text/html;charset=utf-8" }));
  const a = document.createElement("a");
  a.href = url;
  a.download = `Apuntes ${topic.code}.html`;
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

// StudyDocumentPage shows a topic's note as a document to read straight
// through, print or save as PDF, or download.
export function StudyDocumentPage() {
  const { topicId } = useParams();
  const { data: topic, error, loading } = useResource<StudyTopic>(`/study/topics/${topicId}`);
  const [quotes, setQuotes] = useState(() => loadPref(QUOTES_KEY) === "1");
  const article = useRef<HTMLElement>(null);

  // The title names the PDF the browser saves.
  useEffect(() => {
    if (!topic) return;
    const before = document.title;
    document.title = `Apuntes ${topic.code} · TAI Go`;
    return () => {
      document.title = before;
    };
  }, [topic]);

  function toggleQuotes() {
    setQuotes(!quotes);
    savePref(QUOTES_KEY, quotes ? "0" : "1");
  }

  return (
    <div className="doc-page">
      <div className="doc-toolbar">
        <Link to={`/study/${topicId}`} className="doc-back">
          ← Tema
        </Link>
        {topic?.note && (
          <div className="doc-actions">
            <button type="button" className={quotes ? "chip active" : "chip"} aria-pressed={quotes} onClick={toggleQuotes}>
              Citas
            </button>
            <button type="button" className="chip" title="Imprimir o guardar como PDF" onClick={() => window.print()}>
              PDF
            </button>
            <button type="button" className="chip" onClick={() => article.current && void download(topic, article.current)}>
              Descargar
            </button>
          </div>
        )}
      </div>
      <div className="doc-status">
        <ErrorBox message={error} />
        {loading && <Loading />}
        {topic && !topic.note && <p>Este tema todavía no tiene apuntes.</p>}
      </div>
      {topic?.note && (
        <article className="doc" ref={article}>
          <Document topic={topic} note={topic.note} quotes={quotes} />
        </article>
      )}
    </div>
  );
}
