import { Link } from "react-router";
import type { TopicStats } from "../../types";

const roman = ["I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X"];

// BlockProgress shows accuracy per syllabus block, a summary of the
// statistics page.
export function BlockProgress({ topics }: { topics: TopicStats[] }) {
  const blocks: { id: number; name: string; answered: number; correct: number }[] = [];
  for (const t of topics) {
    let b = blocks.find((x) => x.id === t.block_id);
    if (!b) blocks.push((b = { id: t.block_id, name: t.block_name, answered: 0, correct: 0 }));
    b.answered += t.answered;
    b.correct += t.correct;
  }
  if (blocks.length === 0) return null;

  return (
    <section className="home-section">
      <div className="section-head">
        <h3>Por bloque</h3>
        <Link to="/stats" className="small">
          Ver progreso
        </Link>
      </div>
      <ul className="blocks">
        {blocks.map((b, i) => {
          const pct = b.answered > 0 ? Math.round((b.correct / b.answered) * 100) : null;
          return (
            <li key={b.id} className="blk">
              <span className="blk-n">{roman[i] ?? i + 1}</span>
              <span className="blk-t">{b.name}</span>
              <span className="blk-p">{pct === null ? "—" : `${pct} %`}</span>
              <span className="bar" role="img" aria-label={pct === null ? "Sin responder" : `${pct} % de aciertos`}>
                {pct !== null && <i style={{ width: `${pct}%` }} />}
              </span>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
