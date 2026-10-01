import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { api, errorMessage } from "../api";
import { useDebounced, useResource } from "../hooks";
import {
  modeLabel,
  originLabel,
  penaltyOptions,
  type Block,
  type CreateTestInput,
  type Origin,
  type ScoringSettings,
  type Source,
  type TestFilters,
  type TestMode,
} from "../types";
import { ErrorBox, Field } from "../components/Form";
import { TopicPicker } from "../components/TopicPicker";
import { emptyFilters } from "./test/start";

const STORAGE_KEY = "tai.newTest";
const countPresets = [10, 25, 50, 100];
// The real exam allows about 1.2 minutes per question (100 in 120 min).
const minutesFor = (count: number) => Math.max(1, Math.round(count * 1.2));

const defaults: CreateTestInput = {
  mode: "practice",
  filters: emptyFilters,
  count: 25,
  penalty: 1 / 3,
  time_limit_min: minutesFor(25),
};

// Which questions to draw from, on top of the other filters. Maps to the
// due/failed filters; ?set=due|failed preselects one.
type Selection = "all" | "due" | "failed";
const selectionLabel: Record<Selection, string> = { all: "Todas", due: "Repaso de hoy", failed: "Falladas" };
const selectionHint: Record<Selection, string> = {
  all: "Preguntas al azar.",
  due: "Las que te toca repasar hoy según tus respuestas anteriores (repetición espaciada).",
  failed: "Las que fallaste la última vez que te salieron.",
};
const selectionOf = (f: TestFilters): Selection => (f.due ? "due" : f.failed ? "failed" : "all");

// The last configuration is remembered on this device only (without the
// review selection, which changes every day). Returns null when there is
// none, so the scoring defaults from the settings apply.
function loadSaved(): CreateTestInput | null {
  try {
    const saved = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "null");
    if (saved && typeof saved === "object") {
      return { ...defaults, ...saved, filters: { ...emptyFilters, ...saved.filters, question_ids: [], due: false, failed: false } };
    }
  } catch {
    // Storage unavailable or corrupt: start from defaults.
  }
  return null;
}

export function NewTestPage() {
  const navigate = useNavigate();
  const { data: blocks } = useResource<Block[]>("/syllabus");
  const { data: sources } = useResource<Source[]>("/sources");
  const [params] = useSearchParams();
  const [saved] = useState(loadSaved);
  // ?topic=ID (from the syllabus) starts a test on that topic only.
  const [form, setForm] = useState<CreateTestInput>(() => {
    const base = saved ?? defaults;
    const topic = Number(params.get("topic"));
    const set = params.get("set");
    const filters = topic ? { ...emptyFilters, topic_ids: [topic] } : base.filters;
    return { ...base, filters: { ...filters, due: set === "due", failed: set === "failed" } };
  });
  const scoring = useResource<ScoringSettings>(saved ? null : "/settings/scoring");
  useEffect(() => {
    const penalty = scoring.data?.default_penalty;
    if (penalty !== undefined) setForm((f) => ({ ...f, penalty }));
  }, [scoring.data]);
  const [timed, setTimed] = useState(form.time_limit_min > 0);
  const [available, setAvailable] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const debouncedFilters = useDebounced(form.filters, 250);
  useEffect(() => {
    let cancelled = false;
    api<{ available: number }>("/tests/available", { method: "POST", body: debouncedFilters })
      .then((r) => !cancelled && setAvailable(r.available))
      .catch((err) => !cancelled && setError(errorMessage(err)));
    return () => {
      cancelled = true;
    };
  }, [debouncedFilters]);

  const setFilters = (patch: Partial<TestFilters>) => setForm((f) => ({ ...f, filters: { ...f.filters, ...patch } }));

  function setCount(count: number) {
    setForm((f) => ({ ...f, count, time_limit_min: minutesFor(count) }));
  }

  function toggle<T>(list: T[], value: T): T[] {
    return list.includes(value) ? list.filter((v) => v !== value) : [...list, value];
  }

  const citable = sources?.filter((s) => s.kind !== "inap_exam" && s.questions > 0) ?? [];
  const willGet = available === null ? form.count : Math.min(form.count, available);

  async function start() {
    setBusy(true);
    setError(null);
    const body: CreateTestInput = {
      ...form,
      time_limit_min: form.mode === "exam" && timed ? form.time_limit_min : 0,
    };
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(body));
    } catch {
      // Not critical.
    }
    try {
      const { id } = await api<{ id: number }>("/tests", { method: "POST", body });
      navigate(`/tests/${id}`, { replace: true });
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  }

  return (
    <div className="form">
      <h2>Nuevo test</h2>
      <ErrorBox message={error} />

      <fieldset>
        <Field label="Modo">
          <div className="segmented">
            {(Object.keys(modeLabel) as TestMode[]).map((m) => (
              <button
                key={m}
                type="button"
                className={form.mode === m ? "active" : ""}
                onClick={() => setForm((f) => ({ ...f, mode: m }))}
              >
                {modeLabel[m]}
              </button>
            ))}
          </div>
        </Field>
        <p className="hint">
          {form.mode === "practice"
            ? "Corrección inmediata con explicación y fuente tras cada pregunta."
            : "Sin corrección hasta entregar. Puedes cambiar respuestas o dejarlas en blanco."}
        </p>

        <Field label="Número de preguntas">
          <div className="chips">
            {countPresets.map((n) => (
              <button key={n} type="button" className={form.count === n ? "chip active" : "chip"} onClick={() => setCount(n)}>
                {n}
              </button>
            ))}
            <input
              type="number"
              inputMode="numeric"
              min={1}
              max={200}
              className="chip-input"
              aria-label="Otro número"
              value={form.count}
              onChange={(e) => setCount(Math.max(1, Math.min(200, Number(e.target.value) || 1)))}
            />
          </div>
        </Field>

        <Field label="Penalización en la nota">
          <select value={form.penalty} onChange={(e) => setForm((f) => ({ ...f, penalty: Number(e.target.value) }))}>
            {penaltyOptions.map((o) => (
              <option key={o.label} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </Field>

        {form.mode === "exam" && (
          <>
            <label className="check">
              <input type="checkbox" checked={timed} onChange={(e) => setTimed(e.target.checked)} />
              <span>Con tiempo límite</span>
            </label>
            {timed && (
              <Field label="Minutos" hint="El reloj sigue corriendo aunque cierres la app, como en el examen real.">
                <input
                  type="number"
                  inputMode="numeric"
                  min={1}
                  max={300}
                  value={form.time_limit_min}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, time_limit_min: Math.max(1, Math.min(300, Number(e.target.value) || 1)) }))
                  }
                />
              </Field>
            )}
          </>
        )}
      </fieldset>

      <fieldset>
        <legend>Qué preguntas</legend>
        <Field label="Selección">
          <div className="segmented">
            {(Object.keys(selectionLabel) as Selection[]).map((sel) => (
              <button
                key={sel}
                type="button"
                className={selectionOf(form.filters) === sel ? "active" : ""}
                onClick={() => setFilters({ due: sel === "due", failed: sel === "failed" })}
              >
                {selectionLabel[sel]}
              </button>
            ))}
          </div>
        </Field>
        <p className="hint">{selectionHint[selectionOf(form.filters)]}</p>
        <Field label="Temas" hint="Sin seleccionar = todo el temario">
          {blocks && (
            <TopicPicker
              blocks={blocks}
              selected={form.filters.topic_ids}
              onChange={(ids) => setFilters({ topic_ids: ids })}
            />
          )}
        </Field>

        <Field label="Origen" hint="Sin seleccionar = todos">
          <div className="chips">
            {(Object.keys(originLabel) as Origin[]).map((o) => (
              <button
                key={o}
                type="button"
                className={form.filters.origins.includes(o) ? "chip active" : "chip"}
                onClick={() => setFilters({ origins: toggle(form.filters.origins, o) })}
              >
                {originLabel[o]}
              </button>
            ))}
          </div>
        </Field>

        {citable.length > 0 && (
          <Field label="Ley o documento concreto" hint="Sin seleccionar = todos">
            <div className="topic-picker">
              {citable.map((s) => (
                <label key={s.id} className="check">
                  <input
                    type="checkbox"
                    checked={form.filters.source_ids.includes(s.id)}
                    onChange={() => setFilters({ source_ids: toggle(form.filters.source_ids, s.id) })}
                  />
                  <span>
                    {s.title} <span className="muted small">({s.questions})</span>
                  </span>
                </label>
              ))}
            </div>
          </Field>
        )}
      </fieldset>

      <div className="actions sticky">
        <button className="primary" disabled={busy || available === 0} onClick={start}>
          {available === 0
            ? "No hay preguntas con estos filtros"
            : busy
              ? "Preparando…"
              : `Empezar · ${willGet} preguntas`}
        </button>
      </div>
      {available !== null && available > 0 && available < form.count && (
        <p className="hint">Solo hay {available} preguntas publicadas con estos filtros.</p>
      )}
    </div>
  );
}
