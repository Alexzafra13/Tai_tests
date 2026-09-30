import { useState } from "react";
import { api, ApiError, errorMessage, type FieldErrors } from "../../api";
import { roleLabel, type Role, type User } from "../../types";
import { ErrorBox, Field } from "../../components/Form";

// UserCard shows an account and, when expanded, lets an administrator
// change its name, role and status or set a new password.
export function UserCard({ user, isSelf, onChange }: { user: User; isSelf: boolean; onChange: () => void }) {
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ display_name: user.display_name, role: user.role, active: user.active });
  const [password, setPassword] = useState("");
  const [fields, setFields] = useState<FieldErrors>({});
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  async function run(action: () => Promise<unknown>, done: string) {
    setError(null);
    setFields({});
    setNotice(null);
    try {
      await action();
      setNotice(done);
      onChange();
    } catch (err) {
      setError(errorMessage(err));
      if (err instanceof ApiError) setFields(err.fields);
    }
  }

  const save = () => run(() => api(`/users/${user.id}`, { method: "PUT", body: form }), "Guardado");
  const resetPassword = () =>
    run(async () => {
      await api(`/users/${user.id}/password`, { method: "PUT", body: { password } });
      setPassword("");
    }, "Contraseña cambiada; se han cerrado sus sesiones");

  return (
    <li className="card user-card">
      <button type="button" className="user-summary" onClick={() => setOpen((o) => !o)} aria-expanded={open}>
        <span className="row-main">
          <strong>{user.display_name || user.username}</strong>
          <span className="muted small">
            {user.username}
            {isSelf && " · tú"}
          </span>
        </span>
        <span className="meta">
          <span className={user.role === "admin" ? "badge status-reviewed" : "badge"}>{roleLabel[user.role]}</span>
          {!user.active && <span className="badge warn">Desactivado</span>}
        </span>
      </button>

      {open && (
        <div className="user-edit">
          <ErrorBox message={error} />
          {notice && <p className="ok-box">{notice}</p>}
          <Field label="Nombre" htmlFor={`name-${user.id}`}>
            <input
              id={`name-${user.id}`}
              value={form.display_name}
              onChange={(e) => setForm((f) => ({ ...f, display_name: e.target.value }))}
            />
          </Field>
          <Field label="Rol" htmlFor={`role-${user.id}`}>
            <select
              id={`role-${user.id}`}
              value={form.role}
              onChange={(e) => setForm((f) => ({ ...f, role: e.target.value as Role }))}
            >
              {(Object.keys(roleLabel) as Role[]).map((r) => (
                <option key={r} value={r}>
                  {roleLabel[r]}
                </option>
              ))}
            </select>
          </Field>
          <label className="check">
            <input
              type="checkbox"
              checked={form.active}
              onChange={(e) => setForm((f) => ({ ...f, active: e.target.checked }))}
            />
            <span>Activo (puede iniciar sesión)</span>
          </label>
          <div className="actions">
            <button type="button" className="primary" onClick={save}>
              Guardar
            </button>
          </div>

          <Field label="Nueva contraseña" htmlFor={`pw-${user.id}`} error={fields.password}>
            <input
              id={`pw-${user.id}`}
              type="text"
              autoComplete="off"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </Field>
          <div className="actions">
            <button type="button" disabled={!password} onClick={resetPassword}>
              Cambiar contraseña
            </button>
          </div>
        </div>
      )}
    </li>
  );
}
