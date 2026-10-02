import { useState, type FormEvent } from "react";
import { ApiError, errorMessage, type FieldErrors } from "../api";
import { useAuth, type SetupInput } from "../auth";
import { Field } from "../components/Form";
import "./auth.css";

// SetupPage is shown once, on a fresh install, to create the first
// administrator. Everything else is configured from the app afterwards.
export function SetupPage() {
  const { setup } = useAuth();
  const [form, setForm] = useState<SetupInput>({ username: "", display_name: "", password: "" });
  const [repeat, setRepeat] = useState("");
  const [fields, setFields] = useState<FieldErrors>({});
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const set = (key: keyof SetupInput, value: string) => setForm((f) => ({ ...f, [key]: value }));
  const mismatch = repeat !== "" && repeat !== form.password;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (mismatch) return;
    setBusy(true);
    setError(null);
    setFields({});
    try {
      await setup(form);
    } catch (err) {
      setError(errorMessage(err));
      if (err instanceof ApiError) setFields(err.fields);
      setBusy(false);
    }
  }

  return (
    <div className="center">
      <form className="card login setup" onSubmit={onSubmit}>
        <h1>Bienvenido a TAI Go</h1>
        <p className="muted">
          Es la primera vez que se abre esta instalación. Crea la cuenta de administrador: con ella darás de alta al resto
          de usuarios y gestionarás el contenido.
        </p>
        {error && <p className="error">{error}</p>}
        <Field label="Usuario" htmlFor="username" error={fields.username} hint="Minúsculas, números, . _ -">
          <input
            id="username"
            autoComplete="username"
            autoCapitalize="none"
            autoCorrect="off"
            autoFocus
            value={form.username}
            onChange={(e) => set("username", e.target.value)}
            required
          />
        </Field>
        <Field label="Nombre" htmlFor="display_name">
          <input id="display_name" value={form.display_name} onChange={(e) => set("display_name", e.target.value)} />
        </Field>
        <Field label="Contraseña" htmlFor="password" error={fields.password} hint="Mínimo 8 caracteres">
          <input
            id="password"
            type="password"
            autoComplete="new-password"
            value={form.password}
            onChange={(e) => set("password", e.target.value)}
            required
          />
        </Field>
        <Field label="Repite la contraseña" htmlFor="repeat" error={mismatch ? "No coincide" : undefined}>
          <input
            id="repeat"
            type="password"
            autoComplete="new-password"
            value={repeat}
            onChange={(e) => setRepeat(e.target.value)}
            required
          />
        </Field>
        <button className="primary" type="submit" disabled={busy || mismatch || !form.username || !form.password}>
          {busy ? "Creando…" : "Crear administrador y entrar"}
        </button>
      </form>
    </div>
  );
}
