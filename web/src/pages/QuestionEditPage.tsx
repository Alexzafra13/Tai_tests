import { useEffect, useState, type FormEvent } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { api, ApiError, errorMessage, type FieldErrors } from "../api";
import { useResource } from "../hooks";
import {
  authorLabel,
  optionLetters,
  originLabel,
  originSourceKind,
  statusLabel,
  type Block,
  type Origin,
  type Question,
  questionInput,
  type QuestionInput,
  type Source,
  type Status,
} from "../types";
import { ErrorBox, Field, Loading } from "../components/Form";
import { TopicPicker } from "../components/TopicPicker";
import { QuoteHint, useQuoteCheck } from "../components/QuoteCheck";


const empty: QuestionInput = {
  stem: "",
  options: ["", "", "", ""],
  correct: 0,
  explanation: "",
  origin: "law",
  author: "manual",
  source_id: 0,
  source_ref: "",
  source_quote: "",
  status: "published",
  annulled: false,
  fixed_order: false,
  topic_ids: [],
};

export function QuestionEditPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const existing = useResource<Question>(id ? `/questions/${id}` : null);
  const { data: sources } = useResource<Source[]>("/sources");
  const { data: blocks } = useResource<Block[]>("/syllabus");

  const [form, setForm] = useState<QuestionInput>(empty);
  const [fields, setFields] = useState<FieldErrors>({});
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (existing.data) setForm(questionInput(existing.data));
  }, [existing.data]);

  const set = <K extends keyof QuestionInput>(key: K, value: QuestionInput[K]) =>
    setForm((f) => ({ ...f, [key]: value }));

  const setOption = (i: number, value: string) =>
    setForm((f) => {
      const options = [...f.options] as QuestionInput["options"];
      options[i] = value;
      return { ...f, options };
    });

  const setOrigin = (origin: Origin) =>
    setForm((f) => {
      const source = sources?.find((s) => s.id === f.source_id);
      const keepSource = source?.kind === originSourceKind[origin];
      return { ...f, origin, source_id: keepSource ? f.source_id : 0 };
    });

  const eligibleSources = sources?.filter((s) => s.kind === originSourceKind[form.origin]) ?? [];
  const selectedSource = sources?.find((s) => s.id === form.source_id);
  const quoteCheck = useQuoteCheck(selectedSource, form.source_quote);

  async function save(andNew: boolean) {
    setBusy(true);
    setError(null);
    setSaved(null);
    setFields({});
    try {
      if (id) {
        await api(`/questions/${id}`, { method: "PUT", body: form });
        navigate(-1);
        return;
      }
      await api("/questions", { method: "POST", body: form });
      if (andNew) {
        // Keep origin, source and topics: questions are usually entered in
        // batches from the same law or exam.
        setForm((f) => ({ ...empty, origin: f.origin, source_id: f.source_id, topic_ids: f.topic_ids, status: f.status }));
        setSaved("Pregunta guardada. Puedes introducir la siguiente.");
        window.scrollTo(0, 0);
      } else {
        navigate("/questions");
      }
    } catch (err) {
      setError(errorMessage(err));
      if (err instanceof ApiError) setFields(err.fields);
      window.scrollTo(0, 0);
    } finally {
      setBusy(false);
    }
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    save(false);
  }

  async function onDelete() {
    if (!id || !confirm("¿Borrar esta pregunta?")) return;
    try {
      await api(`/questions/${id}`, { method: "DELETE" });
      navigate("/questions");
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  if (existing.loading) return <Loading />;
  if (existing.error) return <ErrorBox message={existing.error} />;

  const needsQuote = form.origin !== "official";

  return (
    <form className="form" onSubmit={onSubmit}>
      <div className="page-head">
        <h2>{id ? `Pregunta #${id}` : "Nueva pregunta"}</h2>
        <button type="button" className="link" onClick={() => navigate(-1)}>
          Cancelar
        </button>
      </div>
      {id && (
        <p className="muted small">
          Autor: {authorLabel[form.author]}
          {existing.data && existing.data.open_reports > 0 && (
            <>
              {" · "}
              <Link to="/review">{existing.data.open_reports} dudas abiertas en Revisión</Link>
            </>
          )}
        </p>
      )}
      <ErrorBox message={error} />
      {saved && <p className="ok-box">{saved}</p>}

      <fieldset>
        <legend>Fuente</legend>
        <Field label="Origen" htmlFor="origin" error={fields.origin}>
          <div className="segmented" id="origin">
            {(Object.keys(originLabel) as Origin[]).map((o) => (
              <button
                key={o}
                type="button"
                className={form.origin === o ? "active" : ""}
                onClick={() => setOrigin(o)}
              >
                {originLabel[o]}
              </button>
            ))}
          </div>
        </Field>

        <Field
          label="Documento"
          htmlFor="source"
          error={fields.source_id}
          hint={
            eligibleSources.length === 0 && sources ? (
              <>
                No hay fuentes de este tipo. <Link to="/sources/new">Crear una</Link>
              </>
            ) : undefined
          }
        >
          <select id="source" value={form.source_id} onChange={(e) => set("source_id", Number(e.target.value))}>
            <option value={0}>Elige la fuente…</option>
            {eligibleSources.map((s) => (
              <option key={s.id} value={s.id}>
                {s.title}
                {s.reference && ` (${s.reference})`}
              </option>
            ))}
          </select>
        </Field>

        <Field
          label={form.origin === "official" ? "Año y nº de pregunta" : "Artículo o apartado"}
          htmlFor="source_ref"
          error={fields.source_ref}
        >
          <input
            id="source_ref"
            value={form.source_ref}
            onChange={(e) => set("source_ref", e.target.value)}
            placeholder={form.origin === "official" ? "2024 · pregunta 37" : "art. 21.2"}
          />
        </Field>

        <Field
          label={needsQuote ? "Cita literal que justifica la respuesta" : "Cita literal (opcional)"}
          htmlFor="source_quote"
          error={fields.source_quote}
          hint={<QuoteHint state={quoteCheck} length={form.source_quote.trim().length} required={needsQuote} />}
        >
          <textarea
            id="source_quote"
            rows={3}
            value={form.source_quote}
            onChange={(e) => set("source_quote", e.target.value)}
            placeholder="Copia aquí el fragmento exacto del texto de la fuente"
          />
        </Field>
      </fieldset>

      <fieldset>
        <legend>Pregunta</legend>
        <Field label="Enunciado" htmlFor="stem" error={fields.stem}>
          <textarea id="stem" rows={3} value={form.stem} onChange={(e) => set("stem", e.target.value)} />
        </Field>

        <div className="field">
          <label>Opciones (marca la correcta)</label>
          {form.options.map((opt, i) => (
            <div key={i} className={`option-edit ${form.correct === i ? "is-correct" : ""}`}>
              <label className="option-radio">
                <input type="radio" name="correct" checked={form.correct === i} onChange={() => set("correct", i)} />
                <span>{optionLetters[i]}</span>
              </label>
              <textarea
                rows={2}
                value={opt}
                aria-label={`Opción ${optionLetters[i]}`}
                onChange={(e) => setOption(i, e.target.value)}
              />
              {fields[`options.${i}`] && <div className="error">{fields[`options.${i}`]}</div>}
            </div>
          ))}
          {fields.correct && <div className="error">{fields.correct}</div>}
        </div>

        <Field label="Explicación" htmlFor="explanation" error={fields.explanation} hint="Basada en la fuente citada">
          <textarea
            id="explanation"
            rows={3}
            value={form.explanation}
            onChange={(e) => set("explanation", e.target.value)}
          />
        </Field>
      </fieldset>

      <fieldset>
        <legend>Clasificación</legend>
        <Field label="Temas" error={fields.topic_ids}>
          {blocks ? (
            <TopicPicker blocks={blocks} selected={form.topic_ids} onChange={(ids) => set("topic_ids", ids)} />
          ) : (
            <Loading />
          )}
        </Field>

        <Field label="Estado" htmlFor="status" error={fields.status}>
          <select id="status" value={form.status} onChange={(e) => set("status", e.target.value as Status)}>
            {(Object.keys(statusLabel) as Status[]).map((s) => (
              <option key={s} value={s}>
                {statusLabel[s]}
              </option>
            ))}
          </select>
        </Field>

        <label className="check">
          <input type="checkbox" checked={form.annulled} onChange={(e) => set("annulled", e.target.checked)} />
          Anulada en el examen oficial
        </label>
        <label className="check">
          <input type="checkbox" checked={form.fixed_order} onChange={(e) => set("fixed_order", e.target.checked)} />
          <span>
            No barajar las opciones
            <span className="hint">
              {" "}
              · Ya se detecta solo cuando una opción nombra letras («A y B») o dice «todas/ninguna de las anteriores».
            </span>
          </span>
        </label>
      </fieldset>

      <div className="actions sticky">
        <button className="primary" type="submit" disabled={busy}>
          {busy ? "Guardando…" : "Guardar"}
        </button>
        {!id && (
          <button type="button" disabled={busy} onClick={() => save(true)}>
            Guardar y otra
          </button>
        )}
        {id && (
          <button type="button" className="danger" onClick={onDelete}>
            Borrar
          </button>
        )}
      </div>
    </form>
  );
}
