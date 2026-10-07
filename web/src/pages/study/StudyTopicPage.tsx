import { Link, useParams } from "react-router";
import { useResource } from "../../hooks";
import type { StudyTopic } from "../../types";
import { ErrorBox, Loading } from "../../components/Form";
import { formatDay, lawOrigin, lawShortName } from "../../format";
import "./study.css";

// StudyTopicPage lists the laws of a topic: the literal consolidated text
// of the BOE, to read with the articles the INAP has asked about.
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
          {topic.laws.length === 0 && (
            <div className="card">
              <p>Este tema todavía no tiene normas para leer.</p>
            </div>
          )}
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
        </>
      )}
    </>
  );
}
