import { useState, type FormEvent } from "react";
import { useAuth } from "../auth";
import { errorMessage } from "../api";
import "./auth.css";
import { Brand } from "../components/Brand";

export function LoginPage() {
  const { login } = useAuth();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await login(username, password);
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  }

  return (
    <div className="center">
      <form className="card login" onSubmit={onSubmit}>
        <h1 className="login-brand">
          <Brand large />
        </h1>
        <p className="muted login-tag">Tu repaso diario para la oposición.</p>
        <label htmlFor="username">Usuario</label>
        <input
          id="username"
          autoComplete="username"
          autoCapitalize="none"
          autoCorrect="off"
          autoFocus
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          required
        />
        <label htmlFor="password">Contraseña</label>
        <input
          id="password"
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />
        {error && <p className="error">{error}</p>}
        <button className="primary" type="submit" disabled={busy || username === "" || password === ""}>
          {busy ? "Entrando…" : "Entrar"}
        </button>
      </form>
    </div>
  );
}
