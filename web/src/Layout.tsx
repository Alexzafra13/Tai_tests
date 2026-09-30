import { Outlet } from "react-router";
import { useAuth } from "./auth";

export function Layout() {
  const { logout } = useAuth();

  return (
    <div className="app">
      <header className="topbar">
        <span className="brand">TAI · Estudio</span>
        <button className="link" onClick={() => logout()}>
          Salir
        </button>
      </header>
      <main className="content">
        <Outlet />
      </main>
    </div>
  );
}
