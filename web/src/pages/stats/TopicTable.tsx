import { useState } from "react";
import { Link } from "react-router";
import type { TopicStats } from "../../types";

type Order = "syllabus" | "weakest" | "exams";
const orderLabel: Record<Order, string> = {
  syllabus: "Temario",
  weakest: "Más flojos",
  exams: "En exámenes",
};

const accuracy = (t: TopicStats) => (t.answered > 0 ? t.correct / t.answered : null);

const sorters: Record<Order, ((a: TopicStats, b: TopicStats) => number) | null> = {
  syllabus: null,
  // Unanswered topics go last: there is nothing to say about them yet.
  weakest: (a, b) => (accuracy(a) ?? 2) - (accuracy(b) ?? 2),
  exams: (a, b) => b.official - a.official,
};

// TopicTable lists every topic with the user's accuracy, how many questions
// there are to practise and how often official exams asked about it.
export function TopicTable({ topics }: { topics: TopicStats[] }) {
  const [order, setOrder] = useState<Order>("syllabus");
  const sorter = sorters[order];
  const rows = sorter ? [...topics].sort(sorter) : topics;

  return (
    <>
      <div className="segmented">
        {(Object.keys(orderLabel) as Order[]).map((o) => (
          <button key={o} type="button" className={o === order ? "active" : ""} onClick={() => setOrder(o)}>
            {orderLabel[o]}
          </button>
        ))}
      </div>
      <ul className="list topic-stats">
        {rows.map((t, i) => {
          const acc = accuracy(t);
          const newBlock = !sorter && (i === 0 || rows[i - 1].block_id !== t.block_id);
          return (
            <li key={t.topic_id}>
              {newBlock && <h4 className="block-name">{t.block_name}</h4>}
              <div className="topic-stat">
                <div className="topic-stat-head">
                  <span>
                    <span className="muted">{t.number}.</span> {t.title}
                  </span>
                  <strong>{acc === null ? "—" : `${Math.round(acc * 100)} %`}</strong>
                </div>
                <div className="meter" role="img" aria-label={acc === null ? "Sin responder" : `${Math.round(acc * 100)} % de aciertos`}>
                  {acc !== null && <span style={{ width: `${acc * 100}%` }} />}
                </div>
                <div className="topic-stat-foot muted small">
                  <span>
                    {t.answered > 0 ? `${t.correct}/${t.answered} aciertos` : "Sin responder"}
                    {t.official > 0 && ` · ${t.official} en exámenes oficiales`}
                  </span>
                  {t.available > 0 && (
                    <Link to={`/tests/new?topic=${t.topic_id}`} className="link">
                      Practicar ({t.available})
                    </Link>
                  )}
                </div>
              </div>
            </li>
          );
        })}
      </ul>
    </>
  );
}
