import { useEffect, useState, type FormEvent } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { api, ApiError, errorMessage, type FieldErrors } from "../api";
import { useResource } from "../hooks";
import { sourceKindLabel, type Source, type SourceInput, type SourceKind } from "../types";
import { ErrorBox, Field, Loading } from "../components/Form";

const empty: SourceInput = { kind: "law", title: "", reference: "", url: "", version_date: "", full_text: "" };

export function SourceEditPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const existing = useResource<Source>(id ? `/sources/${id}` : null);
  const [form, setForm] = useState<SourceInput>(empty);
  const [fields, setFields] = useState<FieldErrors>({});
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    const s = existing.data;
    if (s) {
      setForm({
        kind: s.kind,
        title: s.title,
        reference: s.reference,
        url: s.url,
        version_date: s.version_date,
        full_text: s.full_text ?? "",
      });
    }
  }, [existing.data]);

  const set = <K extends keyof SourceInput>(key: K, value: SourceInput[K]) => setForm((f) => ({ ...f, [key]: value }));

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    setFields({});
    try {
      if (id) await api(`/sources/${id}`, { method: "PUT", body: form });
      else await api("/sources", { method: "POST", body: form });
      navigate("/sources");
    } catch (err) {
      setError(errorMessage(err));
      if (err instanceof ApiError) setFields(err.fields);
    } finally {
      setBusy(false);
    }
  }

  async function onDelete() {
    if (!id || !confirm("¿Borrar esta fuente?")) return;
    try {
      await api(`/sources/${id}`, { method: "DELETE" });
      navigate("/sources");
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  if (existing.loading) return <Loading />;
  if (existing.error) return <ErrorBox message={existing.error} />;

  const words = form.full_text.trim() ? form.full_text.trim().split(/\s+/).length : 0;

  return (
    <form className="form" onSubmit={onSubmit}>
      <div className="page-head">
        <h2>{id ? "Editar fuente" : "Nueva fuente"}</h2>
        <Link to="/sources" className="link">
          Cancelar
        </Link>
      </div>
      <ErrorBox message={error} />

      <Field label="Tipo" htmlFor="kind" error={fields.kind}>
        <select id="kind" value={form.kind} onChange={(e) => set("kind", e.target.value as SourceKind)}>
          {(Object.keys(sourceKindLabel) as SourceKind[]).map((k) => (
            <option key={k} value={k}>
              {sourceKindLabel[k]}
            </option>
          ))}
        </select>
      </Field>

      <Field label="Título" htmlFor="title" error={fields.title}>
        <input
          id="title"
          value={form.title}
          onChange={(e) => set("title", e.target.value)}
          placeholder={
            form.kind === "inap_exam"
              ? "TAI ingreso libre · OEP 2024"
              : "Ley 39/2015, del Procedimiento Administrativo Común"
          }
        />
      </Field>

      <Field
        label="Referencia"
        htmlFor="reference"
        error={fields.reference}
        hint="Identificador oficial: BOE-A-2015-10565, convocatoria y año, RFC 9110…"
      >
        <input id="reference" value={form.reference} onChange={(e) => set("reference", e.target.value)} />
      </Field>

      <Field label="URL" htmlFor="url" error={fields.url}>
        <input id="url" type="url" inputMode="url" value={form.url} onChange={(e) => set("url", e.target.value)} />
      </Field>

      <Field label="Fecha de la versión" htmlFor="version_date" error={fields.version_date} hint="Fecha del texto consolidado">
        <input
          id="version_date"
          type="date"
          value={form.version_date}
          onChange={(e) => set("version_date", e.target.value)}
        />
      </Field>

      <Field
        label="Texto completo"
        htmlFor="full_text"
        error={fields.full_text}
        hint={
          form.kind === "inap_exam"
            ? "Opcional para exámenes."
            : `Necesario para verificar las citas de las preguntas. ${words} palabras.`
        }
      >
        <textarea
          id="full_text"
          rows={10}
          value={form.full_text}
          onChange={(e) => set("full_text", e.target.value)}
        />
      </Field>

      <div className="actions">
        <button className="primary" type="submit" disabled={busy}>
          {busy ? "Guardando…" : "Guardar"}
        </button>
        {id && (
          <button type="button" className="danger" onClick={onDelete}>
            Borrar
          </button>
        )}
      </div>
    </form>
  );
}
