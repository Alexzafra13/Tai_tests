import { useState, type FormEvent } from "react";
import { useAuth } from "../auth";

export function LoginPage() {
  const { login } = useAuth();
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await login(password);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Error inesperado");
      setBusy(false);
    }
  }

  return (
    <div className="center">
      <form className="card login" onSubmit={onSubmit}>
        <h1>TAI · Estudio</h1>
        <label htmlFor="password">Contraseña</label>
        <input
          id="password"
          type="password"
          autoComplete="current-password"
          autoFocus
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />
        {error && <p className="error">{error}</p>}
        <button className="primary" type="submit" disabled={busy || password === ""}>
          {busy ? "Entrando…" : "Entrar"}
        </button>
      </form>
    </div>
  );
}
