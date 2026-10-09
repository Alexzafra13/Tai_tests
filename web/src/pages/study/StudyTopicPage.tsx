import { Link, useParams } from "react-router";
import { useResource } from "../../hooks";
import type { StudyTopic } from "../../types";
import { ErrorBox, Loading } from "../../components/Form";
import { formatDay, lawOrigin, lawShortName } from "../../format";
import { AskedQuestion } from "./LawPage";
import { NoteView } from "./NoteView";
import "./study.css";

// StudyTopicPage lists the laws of a topic: the literal consolidated text
// of the BOE, to read with the articles the INAP has asked about; and the
// official documentation that answers its exam questions, with them.
export function StudyTopicPage() {
  const { topicId } = useParams();
  const { data: topic, error, loading } = useResource<StudyTopic>(`/study/topics/${topicId}`);

  return (
    <>
      <div className="page-head">
        <h2>Estudiar</h2>
        <Link to="/syllabus" className="link">
          Temario
        </Link>
      </div>
      <ErrorBox message={error} />
      {loading && <Loading />}
      {topic && (
        <>
          <p className="muted study-block">{topic.block}</p>
          <h3 className="study-topic">
            {topic.number}. {topic.title}
          </h3>
          {topic.note && (
            <>
              <h3 className="study-section">Apuntes</h3>
              <p className="muted study-note">
                Cada punto lleva el artículo o la página oficial que lo respalda: ábrelo para leer la cita literal. Las
                etiquetas marcan cuántas preguntas de examen ha habido sobre ese artículo.
              </p>
              <NoteView note={topic.note} topicId={topic.topic_id} />
            </>
          )}
          {topic.laws.length === 0 && topic.pages.length === 0 && !topic.note && (
            <div className="card">
              <p>Este tema todavía no tiene apuntes, normas ni documentación para leer.</p>
            </div>
          )}
          {topic.note && topic.laws.length > 0 && <h3 className="study-section">Normas</h3>}
          <ul className="law-list">
            {topic.laws.map((l) => (
              <li key={l.source_id}>
                <Link to={`/study/${topic.topic_id}/laws/${l.source_id}`} className="card law-card">
                  <strong className="law-title">{lawShortName(l.title) || l.title}</strong>
                  {lawShortName(l.title) && <span className="muted law-meta">{l.title}</span>}
                  {l.parts.length > 0 && <span className="law-parts">{l.parts.join(" · ")}</span>}
                  <span className="muted law-meta">
                    {lawOrigin(l.reference).text} · versión del {formatDay(l.version_date)}
                  </span>
                  {l.cited_articles > 0 && (
                    <span className="badge warn">
                      {l.cited_articles === 1 ? "1 artículo preguntado" : `${l.cited_articles} artículos preguntados`} en
                      exámenes
                    </span>
                  )}
                </Link>
              </li>
            ))}
          </ul>
          {topic.pages.length > 0 && (
            <>
              <h3 className="study-section">Documentación oficial</h3>
              <p className="muted study-note">
                Las páginas oficiales que responden a las preguntas de examen del tema, con esas preguntas para
                practicar.
              </p>
              <ul className="law-list">
                {topic.pages.map((p) => (
                  <li key={p.url} className="card doc-card">
                    <a href={p.url} target="_blank" rel="noreferrer" className="link law-title">
                      {p.title} ↗
                    </a>
                    <details className="law-asked">
                      <summary>
                        {p.questions.length === 1 ? "1 pregunta de examen" : `${p.questions.length} preguntas de examen`}
                      </summary>
                      {p.questions.map((q) => (
                        <AskedQuestion key={q.id} q={q} />
                      ))}
                    </details>
                  </li>
                ))}
              </ul>
            </>
          )}
        </>
      )}
    </>
  );
}
