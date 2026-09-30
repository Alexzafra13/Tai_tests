import { Link } from "react-router";
import { authorLabel, originLabel, statusLabel, type Block, type ReviewItem } from "../../types";
import { Options } from "../../components/QuestionView";
import { SourceExcerpt } from "../../components/SourceExcerpt";
import { TopicPicker } from "../../components/TopicPicker";

// ReviewCard shows everything needed to judge a question at a glance:
// why it is in the queue, the question with its answer, and the source
// passage that backs it.
export function ReviewCard({
  item,
  blocks,
  topicIds,
  onTopicsChange,
}: {
  item: ReviewItem;
  blocks: Block[];
  topicIds: number[];
  onTopicsChange: (ids: number[]) => void;
}) {
  const topicName = new Map(blocks.flatMap((b) => b.topics.map((t) => [t.id, `${t.number}. ${t.title}`] as const)));

  return (
    <article className="review-card">
      <div className="meta">
        <span className={`badge status-${item.status}`}>{statusLabel[item.status]}</span>
        <span className="badge">{originLabel[item.origin]}</span>
        <span className="badge outline">{authorLabel[item.author]}</span>
        {item.annulled && <span className="badge warn">Anulada</span>}
      </div>

      {item.reports.length > 0 && (
        <div className="flag-note">
          <strong>⚑ Marcada como dudosa</strong>
          <ul className="report-list">
            {item.reports.map((r) => (
              <li key={r.id}>
                <span className="small">{r.username}</span>
                {r.note && <p>{r.note}</p>}
              </li>
            ))}
          </ul>
        </div>
      )}

      <p className="stem">{item.stem}</p>
      <Options options={item.options} correct={item.correct} />
      {item.explanation && <p className="explanation">{item.explanation}</p>}

      <section className="card source-box">
        <div className="muted small">
          {item.source_title} · {item.source_ref}
        </div>
        {item.excerpt ? (
          <SourceExcerpt excerpt={item.excerpt} />
        ) : item.source_quote ? (
          <p className="error small">La cita no se encuentra en el texto de la fuente.</p>
        ) : (
          <p className="muted small">Sin cita: la respalda la plantilla oficial de respuestas.</p>
        )}
      </section>

      <section className="review-topics">
        {topicIds.length > 0 && (
          <div className="meta">
            {topicIds.map((id) => (
              <span key={id} className="badge outline">
                {topicName.get(id) ?? `Tema ${id}`}
              </span>
            ))}
          </div>
        )}
        {item.topic_ids.length === 0 && (
          <details open={topicIds.length === 0}>
            <summary className="small">
              {topicIds.length === 0 ? "Asigna un tema para poder aceptarla" : "Cambiar temas"}
            </summary>
            <TopicPicker blocks={blocks} selected={topicIds} onChange={onTopicsChange} />
          </details>
        )}
      </section>

      <Link to={`/questions/${item.id}`} className="small">
        Editar en el formulario completo
      </Link>
    </article>
  );
}
