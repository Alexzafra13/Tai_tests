import { useState } from "react";
import type { Block } from "../types";

// TopicPicker is a compact multi-select for syllabus topics, grouped by
// block. Blocks start collapsed except those with a selected topic.
export function TopicPicker({
  blocks,
  selected,
  onChange,
}: {
  blocks: Block[];
  selected: number[];
  onChange: (ids: number[]) => void;
}) {
  const [filter, setFilter] = useState("");
  const chosen = new Set(selected);
  const needle = filter.trim().toLowerCase();

  function toggle(id: number) {
    const next = new Set(chosen);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    onChange([...next]);
  }

  if (blocks.length === 0) {
    return <p className="muted small">No hay temario cargado.</p>;
  }

  return (
    <div className="topic-picker">
      <input type="search" placeholder="Filtrar temas…" value={filter} onChange={(e) => setFilter(e.target.value)} />
      {blocks.map((b) => {
        const topics = needle ? b.topics.filter((t) => t.title.toLowerCase().includes(needle)) : b.topics;
        if (topics.length === 0) return null;
        const count = b.topics.filter((t) => chosen.has(t.id)).length;
        return (
          <details key={b.id} open={needle !== "" || count > 0}>
            <summary>
              {b.name}
              {count > 0 && <span className="badge">{count}</span>}
            </summary>
            {topics.map((t) => (
              <label key={t.id} className="check">
                <input type="checkbox" checked={chosen.has(t.id)} onChange={() => toggle(t.id)} />
                <span>
                  {t.number}. {t.title}
                </span>
              </label>
            ))}
          </details>
        );
      })}
    </div>
  );
}
