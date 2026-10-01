import { useState, type FormEvent } from "react";
import { api, ApiError, errorMessage, type FieldErrors } from "../../api";
import { roleLabel, type Role } from "../../types";
import { ErrorBox, Field } from "../../components/Form";

const empty = { username: "", display_name: "", password: "", role: "user" as Role };

export function NewUserForm({ onCreated }: { onCreated: () => void }) {
  const [form, setForm] = useState(empty);
  const [fields, setFields] = useState<FieldErrors>({});
  const [error, setError] = useState<string | null>(null);

  const set = <K extends keyof typeof empty>(key: K, value: (typeof empty)[K]) => setForm((f) => ({ ...f, [key]: value }));

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setFields({});
    try {
      await api("/users", { method: "POST", body: form });
      setForm(empty);
      onCreated();
    } catch (err) {
      setError(errorMessage(err));
      if (err instanceof ApiError) setFields(err.fields);
    }
  }

  return (
    <form onSubmit={onSubmit}>
      <fieldset>
        <legend>Nuevo usuario</legend>
        <ErrorBox message={error} />
        <Field label="Usuario" htmlFor="new-username" error={fields.username} hint="Con esto inicia sesión: minúsculas, números, . _ -">
          <input
            id="new-username"
            autoCapitalize="none"
            autoCorrect="off"
            value={form.username}
            onChange={(e) => set("username", e.target.value)}
          />
        </Field>
        <Field label="Nombre" htmlFor="new-name">
          <input id="new-name" value={form.display_name} onChange={(e) => set("display_name", e.target.value)} />
        </Field>
        <Field label="Contraseña inicial" htmlFor="new-password" error={fields.password} hint="Mínimo 8 caracteres. Podrá cambiarla en Ajustes.">
          <input
            id="new-password"
            type="text"
            autoComplete="off"
            value={form.password}
            onChange={(e) => set("password", e.target.value)}
          />
        </Field>
        <Field label="Rol" htmlFor="new-role" error={fields.role}>
          <select id="new-role" value={form.role} onChange={(e) => set("role", e.target.value as Role)}>
            {(Object.keys(roleLabel) as Role[]).map((r) => (
              <option key={r} value={r}>
                {roleLabel[r]}
              </option>
            ))}
          </select>
        </Field>
        <div className="actions">
          <button type="submit" className="primary" disabled={!form.username || !form.password}>
            Crear usuario
          </button>
        </div>
      </fieldset>
    </form>
  );
}
