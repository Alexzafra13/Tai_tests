import { Fragment, useState } from "react";
import { Link } from "react-router";
import { api, errorMessage } from "../../api";
import type { NotePoint, NoteRef, TopicNote } from "../../types";
import "../../components/QuestionView.css";

function Strong({ text }: { text: string }) {
  return (
    <>
      {text.split("**").map((part, i) => (i % 2 === 1 ? <strong key={i}>{part}</strong> : <Fragment key={i}>{part}</Fragment>))}
    </>
  );
}

// Bold renders the **marked** words of a note's text, and `code` (SQL,
// program fragments) as code, where asterisks are literal.
export function Bold({ text }: { text: string }) {
  return (
    <>
      {text.split("`").map((part, i) => (i % 2 === 1 ? <code key={i}>{part}</code> : <Strong key={i} text={part} />))}
    </>
  );
}

// Ref shows where a point comes from; opening it shows the literal quote.
function Ref({ r }: { r: NoteRef }) {
  return (
    <details className="note-ref">
      <summary>
        {r.label}
        {r.asked > 0 && <span className="badge warn">{r.asked === 1 ? "1 pregunta" : `${r.asked} preguntas`}</span>}
      </summary>
      {r.quotes.map((q, i) => (
        <blockquote key={i}>«{q}»</blockquote>
      ))}
      {r.url ? (
        <a href={r.url} target="_blank" rel="noreferrer" className="link">
          Abrir la fuente ↗
        </a>
      ) : (
        <Link to={`/laws/${r.source_id}#${r.block_id}`} className="link">
          Leer el artículo
        </Link>
      )}
    </details>
  );
}

// ReportPoint lets a user tell the administrators that a point is wrong or
// incomplete.
function ReportPoint({ topicId, point }: { topicId: number; point: NotePoint }) {
  const [open, setOpen] = useState(false);
  const [note, setNote] = useState("");
  const [state, setState] = useState<"idle" | "sending" | "sent">("idle");
  const [error, setError] = useState<string | null>(null);

  async function send() {
    setState("sending");
    setError(null);
    try {
      await api(`/study/topics/${topicId}/report`, {
        method: "POST",
        body: { point: point.path, excerpt: point.text.replaceAll("**", "").replaceAll("`", ""), note },
      });
      setState("sent");
      setOpen(false);
    } catch (err) {
      setError(errorMessage(err));
      setState("idle");
    }
  }

  if (state === "sent") return <p className="muted note-sent">Gracias, lo revisaremos.</p>;
  if (!open) {
    return (
      <button type="button" className="link note-flag" onClick={() => setOpen(true)}>
        ⚐ Avisar de un error
      </button>
    );
  }
  return (
    <div className="flag-editor">
      <textarea
        rows={2}
        autoFocus
        placeholder="¿Qué está mal o qué falta?"
        value={note}
        onChange={(e) => setNote(e.target.value)}
      />
      {error && <div className="error">{error}</div>}
      <div className="actions">
        <button type="button" className="primary" disabled={state === "sending"} onClick={send}>
          Enviar
        </button>
        <button type="button" className="link" onClick={() => setOpen(false)}>
          Cancelar
        </button>
      </div>
    </div>
  );
}

function Point({ p, topicId, top }: { p: NotePoint; topicId: number; top: boolean }) {
  return (
    <li className="note-point">
      <p>
        <Bold text={p.text} />
      </p>
      <div className="note-refs">
        {p.refs.map((r, i) => (
          <Ref key={i} r={r} />
        ))}
      </div>
      {p.items.length > 0 && (
        <ul className="note-items">
          {p.items.map((it) => (
            <Point key={it.path} p={it} topicId={topicId} top={false} />
          ))}
        </ul>
      )}
      {top && <ReportPoint topicId={topicId} point={p} />}
    </li>
  );
}

// NoteView shows a topic's study note: each point with the articles or
// pages that back it, and how often the INAP has asked about them.
export function NoteView({ note, topicId }: { note: TopicNote; topicId: number }) {
  return (
    <div className="note">
      {note.sections.map((s, i) => (
        <section key={i} className="card note-section">
          <h3>{s.title}</h3>
          <ul className="note-points">
            {s.points.map((p) => (
              <Point key={p.path} p={p} topicId={topicId} top />
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}
