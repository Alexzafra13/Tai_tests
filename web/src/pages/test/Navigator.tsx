import type { Test } from "../../types";

// Navigator is the grid of question numbers to jump around a test.
export function Navigator({ test, current, onGo }: { test: Test; current: number; onGo: (i: number) => void }) {
  return (
    <div className="navigator card">
      {test.items.map((it, i) => {
        let cls = "nav-cell";
        if (it.solution) cls += it.solution.is_correct ? " is-correct" : " is-wrong";
        else if (it.chosen !== null) cls += " is-answered";
        if (it.reported) cls += " is-flagged";
        if (i === current) cls += " is-current";
        return (
          <button key={i} type="button" className={cls} onClick={() => onGo(i)}>
            {i + 1}
          </button>
        );
      })}
      <p className="muted small legend">
        <span className="nav-cell is-answered" /> respondida <span className="nav-cell is-flagged" /> dudosa
      </p>
    </div>
  );
}
