import { useState, type FormEvent } from "react";
import { api, ApiError, errorMessage, type FieldErrors } from "../../api";
import { useAuth } from "../../auth";
import { roleLabel } from "../../types";
import { ErrorBox, Field } from "../../components/Form";

export function AccountSection() {
  const { user } = useAuth();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [fields, setFields] = useState<FieldErrors>({});
  const [error, setError] = useState<string | null>(null);
  const [ok, setOk] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setFields({});
    setOk(false);
    try {
      await api("/account/password", { method: "PUT", body: { current_password: current, new_password: next } });
      setCurrent("");
      setNext("");
      setOk(true);
    } catch (err) {
      setError(errorMessage(err));
      if (err instanceof ApiError) setFields(err.fields);
    }
  }

  if (!user) return null;
  return (
    <form onSubmit={onSubmit}>
      <fieldset>
        <legend>Cuenta</legend>
        <p>
          <strong>{user.display_name || user.username}</strong>
          <span className="muted">
            {" "}
            · {user.username} · {roleLabel[user.role]}
          </span>
        </p>
        <ErrorBox message={error} />
        {ok && <p className="ok-box">Contraseña cambiada. Se han cerrado tus otras sesiones.</p>}
        <Field label="Contraseña actual" htmlFor="current_password" error={fields.current_password}>
          <input
            id="current_password"
            type="password"
            autoComplete="current-password"
            value={current}
            onChange={(e) => setCurrent(e.target.value)}
          />
        </Field>
        <Field label="Nueva contraseña" htmlFor="new_password" error={fields.password} hint="Mínimo 8 caracteres">
          <input
            id="new_password"
            type="password"
            autoComplete="new-password"
            value={next}
            onChange={(e) => setNext(e.target.value)}
          />
        </Field>
        <div className="actions">
          <button type="submit" disabled={!current || !next}>
            Cambiar contraseña
          </button>
        </div>
      </fieldset>
    </form>
  );
}
