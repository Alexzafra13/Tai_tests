import { useState } from "react";
import type { StatsDay } from "../../types";

const dayFormat = new Intl.DateTimeFormat("es-ES", { day: "numeric", month: "short", timeZone: "UTC" });
const formatDay = (date: string) => dayFormat.format(new Date(`${date}T00:00:00Z`));

// tipPosition centres the tooltip over its column, anchoring it to the
// column's side near either end so it never leaves the chart.
function tipPosition(i: number, n: number) {
  const left = `${((i + 0.5) / n) * 100}%`;
  if (i < n / 4) return { left, transform: "translateX(-1rem)" };
  if (i >= n - n / 4) return { left, transform: "translateX(calc(-100% + 1rem))" };
  return { left };
}

// ActivityChart shows answers per day as stacked columns: right answers at
// the base, wrong or blank on top. Hovering (or tapping) a day shows its
// numbers; the same data is available as a table below the chart.
export function ActivityChart({ days }: { days: StatsDay[] }) {
  const [active, setActive] = useState<number | null>(null);
  const max = Math.max(1, ...days.map((d) => d.answered));
  const shown = active !== null ? days[active] : null;

  return (
    <figure className="chart">
      <figcaption className="chart-head">
        <span className="chart-legend">
          <span className="swatch series-1" /> Aciertos
        </span>
        <span className="chart-legend">
          <span className="swatch series-2" /> Fallos y en blanco
        </span>
      </figcaption>

      <div className="chart-plot" onMouseLeave={() => setActive(null)}>
        <span className="chart-max muted small">{max}</span>
        <div className="chart-columns">
          {days.map((d, i) => {
            const failed = d.answered - d.correct;
            return (
              <button
                key={d.date}
                type="button"
                className={i === active ? "chart-col active" : "chart-col"}
                aria-label={`${formatDay(d.date)}: ${d.correct} aciertos, ${failed} fallos o en blanco`}
                onMouseEnter={() => setActive(i)}
                onFocus={() => setActive(i)}
                onClick={() => setActive(i)}
              >
                <span className="chart-stack" style={{ height: `${(d.answered / max) * 100}%` }}>
                  {failed > 0 && <span className="seg series-2" style={{ flexGrow: failed }} />}
                  {d.correct > 0 && <span className="seg series-1" style={{ flexGrow: d.correct }} />}
                </span>
              </button>
            );
          })}
        </div>
        {shown && (
          <div
            className="chart-tip"
            style={tipPosition(active!, days.length)}
            role="status"
          >
            <strong>{formatDay(shown.date)}</strong>
            <span>
              <span className="swatch series-1" /> {shown.correct} aciertos
            </span>
            <span>
              <span className="swatch series-2" /> {shown.answered - shown.correct} fallos
            </span>
          </div>
        )}
      </div>
      <div className="chart-axis muted small">
        <span>{formatDay(days[0].date)}</span>
        <span>Hoy</span>
      </div>

      <details className="chart-table">
        <summary className="muted small">Ver como tabla</summary>
        <table>
          <thead>
            <tr>
              <th>Día</th>
              <th>Respondidas</th>
              <th>Aciertos</th>
            </tr>
          </thead>
          <tbody>
            {days
              .filter((d) => d.answered > 0)
              .reverse()
              .map((d) => (
                <tr key={d.date}>
                  <td>{formatDay(d.date)}</td>
                  <td>{d.answered}</td>
                  <td>{d.correct}</td>
                </tr>
              ))}
          </tbody>
        </table>
      </details>
    </figure>
  );
}
