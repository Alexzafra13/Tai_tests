import { useEffect, useState, type FormEvent } from "react";
import { api, ApiError, errorMessage, type FieldErrors } from "../api";
import { useResource } from "../hooks";
import { penaltyOptions, type ScoringSettings } from "../types";
import { ErrorBox, Field, Loading } from "../components/Form";

export function SettingsPage() {
  const saved = useResource<ScoringSettings>("/settings/scoring");
  const [form, setForm] = useState<ScoringSettings>();
  const [fields, setFields] = useState<FieldErrors>({});
  const [error, setError] = useState<string | null>(null);
  const [ok, setOk] = useState(false);

  useEffect(() => {
    if (saved.data) setForm(saved.data);
  }, [saved.data]);

  if (saved.loading || !form) return <Loading />;

  const set = <K extends keyof ScoringSettings>(key: K, value: ScoringSettings[K]) => {
    setOk(false);
    setForm((f) => f && { ...f, [key]: value });
  };

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setFields({});
    try {
      await api("/settings/scoring", { method: "PUT", body: form });
      setOk(true);
    } catch (err) {
      setError(errorMessage(err));
      if (err instanceof ApiError) setFields(err.fields);
    }
  }

  return (
    <form className="form" onSubmit={onSubmit}>
      <h2>Ajustes</h2>
      <ErrorBox message={error ?? saved.error} />
      {ok && <p className="ok-box">Guardado. Los resultados anteriores se muestran ya con la nueva escala.</p>}

      <fieldset>
        <legend>Nota</legend>
        <p className="hint">
          Ajústalo a las bases de tu convocatoria. La nota es (aciertos − fallos × penalización) / preguntas × puntuación
          máxima.
        </p>
        <Field label="Puntuación máxima" htmlFor="max" error={fields.max}>
          <input
            id="max"
            type="number"
            inputMode="decimal"
            step="any"
            value={form.max}
            onChange={(e) => set("max", Number(e.target.value))}
          />
        </Field>
        <Field label="Nota para aprobar" htmlFor="pass_mark" error={fields.pass_mark}>
          <input
            id="pass_mark"
            type="number"
            inputMode="decimal"
            step="any"
            value={form.pass_mark}
            onChange={(e) => set("pass_mark", Number(e.target.value))}
          />
        </Field>
        <Field label="Penalización por defecto" htmlFor="default_penalty" error={fields.default_penalty}>
          <select
            id="default_penalty"
            value={penaltyOptions.find((o) => Math.abs(o.value - form.default_penalty) < 1e-6)?.value ?? form.default_penalty}
            onChange={(e) => set("default_penalty", Number(e.target.value))}
          >
            {penaltyOptions.map((o) => (
              <option key={o.label} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </Field>
      </fieldset>

      <div className="actions">
        <button className="primary" type="submit">
          Guardar
        </button>
      </div>
    </form>
  );
}
